package filelock_test

import (
	"os/exec"

	"garden-external-networker/filelock"

	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Locking using a file", func() {
	var path string
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		path = filepath.Join(tempDir, "dir1", "dir2", "some-file.json")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	AssertBasicThingsWork := func(expectedInitialContents []byte) {
		It("returns a file usable for read and write", func() {
			lock := filelock.Locker{
				Path: path,
			}

			file, err := lock.Open()
			Expect(err).NotTo(HaveOccurred())

			initialContents, err := ioutil.ReadAll(file)
			Expect(err).NotTo(HaveOccurred())

			Expect(initialContents).To(Equal(expectedInitialContents))

			Expect(file.Truncate(0)).To(Succeed())

			_, err = file.Seek(0, 0)
			Expect(err).NotTo(HaveOccurred())

			_, err = file.Write([]byte("hello"))
			Expect(err).NotTo(HaveOccurred())

			_, err = file.Seek(0, 0)
			Expect(err).NotTo(HaveOccurred())

			allBytes, err := ioutil.ReadAll(file)
			Expect(err).NotTo(HaveOccurred())

			Expect(allBytes).To(Equal([]byte("hello")))

			Expect(file.Close()).To(Succeed())

			allBytes, err = ioutil.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(allBytes).To(Equal([]byte("hello")))
		})
	}

	Context("when the file does not already exist", func() {
		BeforeEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})
		AssertBasicThingsWork([]byte{})
	})

	Context("when the file already exists", func() {
		var preExistingContents = []byte("foo bar baz this is some prior data")

		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Dir(path), 0700)).To(Succeed())
			Expect(ioutil.WriteFile(path, preExistingContents, 0600)).To(Succeed())
		})
		AssertBasicThingsWork(preExistingContents)
	})

	Context("when the path has already been opened by a locker in the same OS process", func() {
		var theFirstFileHandle *os.File

		BeforeEach(func() {
			By("acquiring the first lock on the file")
			var err error
			theFirstFileHandle, err = (&filelock.Locker{Path: path}).Open()
			Expect(err).NotTo(HaveOccurred())

			By("writing some data to the file")
			theFirstFileHandle.Write([]byte("the first data"))
		})

		It("blocks the second open until the first one is closed", func(done Done) {
			locker := &filelock.Locker{Path: path}

			By("attempting to acquire another lock on the same file")
			lockAcquiredChan := make(chan struct{})
			var secondFileHandle *os.File
			go func() {
				defer GinkgoRecover()

				var err error
				secondFileHandle, err = locker.Open()
				Expect(err).NotTo(HaveOccurred())

				lockAcquiredChan <- struct{}{}
			}()
			By("verifying that we cannot acquire the lock")
			Consistently(lockAcquiredChan).ShouldNot(Receive())

			By("releasing the first lock")
			Expect(theFirstFileHandle.Close()).To(Succeed())

			By("checking that we can now acquire the second lock on the file")
			Eventually(lockAcquiredChan).Should(Receive())

			By("validating the data written to the first lock")
			Expect(ioutil.ReadAll(secondFileHandle)).To(Equal([]byte("the first data")))

			close(done)
		}, 5 /* max seconds allowed for this spec */)
	})

	Context("when the file is locked from a separate OS process", func() {
		It("blocks the second file open until after the other process has released the lock", func(done Done) {
			cmd := exec.Command(pathToBinary, path)
			stdinPipe, err := cmd.StdinPipe()
			Expect(err).NotTo(HaveOccurred())

			By("starting an external process that will acquire the lock")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for it to acquire the lock")
			Eventually(session.Err).Should(gbytes.Say("done after"))
			Consistently(session).ShouldNot(gexec.Exit())
			_, err = stdinPipe.Write([]byte("some data from the external process"))
			Expect(err).NotTo(HaveOccurred())

			By("attempting to acquire the lock ourselves")
			locker := &filelock.Locker{Path: path}

			lockAcquiredChan := make(chan struct{})
			var file *os.File
			go func() {
				defer GinkgoRecover()

				var err error
				file, err = locker.Open()
				Expect(err).NotTo(HaveOccurred())

				lockAcquiredChan <- struct{}{}
			}()

			By("verifying that we cannot acquire the lock")
			Consistently(lockAcquiredChan).ShouldNot(Receive())

			By("signaling for the external process to release the lock")
			Expect(stdinPipe.Close()).To(Succeed())

			By("checking that we can now acquire the lock")
			Eventually(lockAcquiredChan).Should(Receive())

			By("validating the data written by the external process")
			Expect(ioutil.ReadAll(file)).To(Equal([]byte("some data from the external process")))

			By("releasing the lock")
			Expect(file.Close()).To(Succeed())
			close(done)
		}, 5 /* max seconds allowed for this spec */)
	})
})
