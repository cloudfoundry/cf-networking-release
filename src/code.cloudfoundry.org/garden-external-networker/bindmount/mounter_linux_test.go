package bindmount_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"code.cloudfoundry.org/garden-external-networker/bindmount"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func sameFile(path1, path2 string) bool {
	fi1, err := os.Stat(path1)
	Expect(err).NotTo(HaveOccurred())

	fi2, err := os.Stat(path2)
	Expect(err).NotTo(HaveOccurred())
	return os.SameFile(fi1, fi2)
}

func getInode(path string) uint64 {
	stat := &unix.Stat_t{}
	err := unix.Stat(path, stat)
	Expect(err).NotTo(HaveOccurred())
	return stat.Ino
}

var _ = Describe("Mounter", func() {
	var (
		mounter                                      *bindmount.Mounter
		sourceDir, targetDir, sourceFile, targetFile string
	)

	BeforeEach(func() {
		mounter = &bindmount.Mounter{}

		var err error
		sourceDir, err = ioutil.TempDir("", "bind-mount-test-source-")
		Expect(err).NotTo(HaveOccurred())

		targetDir, err = ioutil.TempDir("", "bind-mount-test-target-")
		Expect(err).NotTo(HaveOccurred())

		sourceFile = filepath.Join(sourceDir, "the-source")
		targetFile = filepath.Join(targetDir, "some-sub-dir", "the-target")

		Expect(ioutil.WriteFile(sourceFile, []byte("some data"), 0644)).To(Succeed())
	})

	Describe("IdempotentlyMount", func() {
		It("should mount the provided source to the target", func() {
			Expect(mounter.IdempotentlyMount(sourceFile, targetFile)).To(Succeed())

			Expect(targetFile).To(BeAnExistingFile())

			Expect(sameFile(sourceFile, targetFile)).To(BeTrue())
		})

		Context("when run repeatedly with the same input", func() {
			It("should behave identically", func() {
				for i := 0; i < 4; i++ {
					Expect(mounter.IdempotentlyMount(sourceFile, targetFile)).To(Succeed())

					Expect(targetFile).To(BeAnExistingFile())
					Expect(sameFile(sourceFile, targetFile)).To(BeTrue())
				}
			})
		})

		Context("when the source filepath is later removed", func() {
			It("should not impact the contents of the target file", func() {
				Expect(mounter.IdempotentlyMount(sourceFile, targetFile)).To(Succeed())

				Expect(os.RemoveAll(sourceDir)).To(Succeed())

				Expect(ioutil.ReadFile(targetFile)).To(Equal([]byte("some data")))
			})

			XIt("should not impact the identity of the target mount point", func() {
				// TODO: figure out why this test fails
				sourceInode := getInode(sourceFile)

				Expect(mounter.IdempotentlyMount(sourceFile, targetFile)).To(Succeed())

				Expect(os.RemoveAll(sourceDir)).To(Succeed())

				targetInode := getInode(targetFile)
				Expect(targetInode).To(Equal(sourceInode))
			})
		})

		Context("when things don't work the way you expect", func() {
			Context("when mkdirall fails", func() {
				It("should return the error", func() {
					brokenTarget := "/proc/0/foo/bar"
					err := mounter.IdempotentlyMount(sourceFile, brokenTarget)
					Expect(err).To(MatchError("os.MkdirAll failed: mkdir /proc/0: no such file or directory"))
				})
			})

			Context("when os.Create fails", func() {
				It("should return the error", func() {
					brokenTarget := targetDir
					err := mounter.IdempotentlyMount(sourceFile, brokenTarget)
					Expect(err).To(MatchError(ContainSubstring("is a directory")))
					Expect(err).To(MatchError(HavePrefix("os.Create failed:")))
				})
			})

			Context("when unix.Mount fails", func() {
				It("should return the error", func() {
					brokenSource := "/proc/-1/foo"
					err := mounter.IdempotentlyMount(brokenSource, targetFile)
					Expect(err).To(MatchError("mount failed: no such file or directory"))
				})
			})
		})
	})

	Describe("RemoveMount", func() {
		It("should idempotently unmount the thing", func() {
			Expect(mounter.IdempotentlyMount(sourceFile, targetFile)).To(Succeed())

			Expect(targetFile).To(BeAnExistingFile())

			Expect(mounter.RemoveMount(targetFile)).To(Succeed())

			Expect(targetFile).NotTo(BeAnExistingFile())
			Expect(targetDir).To(BeADirectory())

			err := mounter.RemoveMount(targetFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(targetFile).NotTo(BeAnExistingFile())
			Expect(targetDir).To(BeADirectory())
		})
	})
})
