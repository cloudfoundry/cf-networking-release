package rotatablesink

import (
	"code.cloudfoundry.org/lager"
	"io"
	"os"
	"time"
	"sync"
	"fmt"
	"syscall"
)

type RotatableSink struct {
	fileToWatch         string
	fileToWatchInode    uint64
	minLogLevel         lager.LogLevel
	WriterFactory       FileWriterFactory
	writerSink          lager.Sink
	writeL              *sync.Mutex
	DestinationFileInfo DestinationFileInfo
}

func (rs *RotatableSink) Log(logFmt lager.LogFormat) {
	rs.writeL.Lock()
	defer rs.writeL.Unlock()
	rs.writerSink.Log(logFmt)
}

func NewRotatableSink(fileToWatch string, logLevel lager.LogLevel, fileWriterFactory FileWriterFactory, destinationFileInfo DestinationFileInfo, componentLogger lager.Logger) (*RotatableSink, error) {
	var err error
	rotatableSink := &RotatableSink{
		fileToWatch:         fileToWatch,
		minLogLevel:         logLevel,
		WriterFactory:       fileWriterFactory,
		DestinationFileInfo: destinationFileInfo,
		writeL:              new(sync.Mutex),
	}

	err = rotatableSink.registerFileSink(fileToWatch)
	if err != nil {
		return nil, fmt.Errorf("register file sink: %s", err)
	}

	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				fileExists, err := destinationFileInfo.FileExists(fileToWatch)
				if err != nil {
					componentLogger.Error("stat-file", fmt.Errorf("stat file: %s", err))
					continue
				}

				if !fileExists {
					err = rotatableSink.registerFileSink(fileToWatch)
					if err != nil {
						componentLogger.Error("register-moved-file-sink", err)
					}
				} else {
					fileToWatchStatInode, err := destinationFileInfo.FileInode(fileToWatch)
					if err != nil {
						componentLogger.Error("register-rotated-file-sink", err)
						continue
					}
					if fileToWatchStatInode != rotatableSink.fileToWatchInode {
						err = rotatableSink.registerFileSink(fileToWatch)
						if err != nil {
							componentLogger.Error("register-rotated-file-sink", err)
						}
					}
				}
			}
		}
	}()

	return rotatableSink, nil
}

func (rs *RotatableSink) registerFileSink(fileToWatch string) error {
	var err error
	err = rs.rotateFileSink()
	if err != nil {
		return fmt.Errorf("rotate file sink: %s", err)
	}

	rs.fileToWatchInode, err = rs.DestinationFileInfo.FileInode(fileToWatch)
	if err != nil {
		return fmt.Errorf("get file inode: %s", err)
	}
	return nil
}

func (rs *RotatableSink) rotateFileSink() error {
	rs.writeL.Lock()
	defer rs.writeL.Unlock()
	outputLogFile, err := rs.WriterFactory.NewWriter(rs.fileToWatch)
	if err != nil {
		return fmt.Errorf("create file writer: %s", err)
	}
	rs.writerSink = lager.NewWriterSink(outputLogFile, rs.minLogLevel)
	return nil
}

type FileWriterFactory interface {
	NewWriter(fileName string) (io.Writer, error)
}

type DefaultFileWriterFunc func(string) (io.Writer, error)

func (dfwf DefaultFileWriterFunc) NewWriter(fileName string) (io.Writer, error) {
	return dfwf(fileName)
}

func DefaultFileWriter(fileName string) (io.Writer, error) {
	return os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
}

//go:generate counterfeiter -o ../fakes/destinationfileinfo.go --fake-name DestinationFileInfo . DestinationFileInfo
type DestinationFileInfo interface {
	FileExists(string) (bool, error)
	FileInode(string) (uint64, error)
}

type DefaultDestinationFileInfo struct{}

func (DefaultDestinationFileInfo) FileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("stat file: %s", err)
	}
	return true, nil

}

func (DefaultDestinationFileInfo) FileInode(filename string) (uint64, error) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return 0, fmt.Errorf("stat file: %s", err)
	}

	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		return stat.Ino, err
	}

	return 0, fmt.Errorf("unable to stat file: %s", err)
}
