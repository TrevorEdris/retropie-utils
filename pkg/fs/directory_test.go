package fs_test

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

var _ = Describe("Directory", func() {
	var (
		tempDir = filepath.Join(os.TempDir(), uuid.New().String())
	)

	BeforeEach(func() {
		Expect(tempDir).NotTo(BeEmpty())
	})

	When("the directory is flat", func() {
		var (
			dir        = filepath.Join(tempDir, "flat")
			addedFiles []string
		)

		BeforeEach(func() {
			err := os.MkdirAll(dir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			addedFiles = []string{
				"aaaa.txt",
				"bbbb.csv",
				"cccc.state",
				"dddd.state1",
				"eeee.sav",
				"ffff.gb",
			}
			for _, file := range addedFiles {
				_, err := os.Create(filepath.Join(dir, file))
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			err := os.RemoveAll(dir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("lists all files in the directory", func() {
			d, err := fs.NewDirectory(dir)
			Expect(err).NotTo(HaveOccurred())
			Expect(d.GetAbsolutePath()).To(Equal(dir))
			Expect(d.GetName()).To(Equal("flat"))
			files := d.GetAllFiles()
			Expect(files).To(HaveLen(6))
		})

		It("gets matching files", func() {
			d, err := fs.NewDirectory(dir)
			Expect(err).NotTo(HaveOccurred())
			Expect(d.GetAbsolutePath()).To(Equal(dir))
			matchingFiles, err := d.GetMatchingFiles(fs.Rom)
			Expect(err).NotTo(HaveOccurred())
			Expect(matchingFiles).To(HaveLen(1))
			Expect(matchingFiles[0].Name).To(Equal("ffff.gb"))
			Expect(matchingFiles[0].Absolute).To(Equal(
				filepath.Join(dir, "ffff.gb"),
			))
			Expect(matchingFiles[0].Dir).To(Equal(dir))
		})
	})

	When("subdirectories exist", func() {

		var (
			dir        = filepath.Join(tempDir, "nonflat")
			addedFiles []string
			subDirs    []string
		)

		BeforeEach(func() {
			err := os.MkdirAll(dir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
			addedFiles = []string{
				"aaaa.txt",
				"bbbb.csv",
				"cccc.state",
				"dddd.state1",
				"eeee.sav",
				"ffff.gb",
			}
			subDirs = []string{
				"subA", "subB", "subC",
			}
			for _, subdir := range subDirs {
				subdir = filepath.Join(dir, subdir)
				err := os.MkdirAll(subdir, os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
				for _, file := range addedFiles {
					_, err = os.Create(filepath.Join(subdir, file))
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})

		AfterEach(func() {
			err := os.RemoveAll(dir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("lists all files in subdirectories also", func() {
			d, err := fs.NewDirectory(dir)
			Expect(err).NotTo(HaveOccurred())
			Expect(d.GetAbsolutePath()).To(Equal(dir))
			Expect(d.GetName()).To(Equal("nonflat"))
			files := d.GetAllFiles()
			Expect(files).To(HaveLen(len(subDirs) * len(addedFiles)))
		})

		It("gets matching files", func() {
			d, err := fs.NewDirectory(dir)
			Expect(err).NotTo(HaveOccurred())
			Expect(d.GetAbsolutePath()).To(Equal(dir))
			matchingFiles, err := d.GetMatchingFiles(fs.Rom)
			Expect(err).NotTo(HaveOccurred())
			Expect(matchingFiles).To(HaveLen(3))
			for _, mf := range matchingFiles {
				Expect(mf.Name).To(Equal("ffff.gb"))
				Expect(mf.Dir).To(ContainSubstring(filepath.Join(dir, "sub")))
			}
		})
	})
})
