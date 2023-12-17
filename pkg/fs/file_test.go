package fs_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

var _ = Describe("File", func() {
	It("parses FileType correctly", func() {
		namesToType := map[string]fs.FileType{
			"aaaa.gb":    fs.Rom,
			"bbbb.sav":   fs.Save,
			"cccc.state": fs.State,
			"dddd.txt":   fs.Other,
		}
		files := make([]*fs.File, 0)
		for filename, _ := range namesToType {
			f := fs.NewFile(filename, time.Now())
			files = append(files, f)
		}
		for _, f := range files {
			expectedType := namesToType[f.Name]
			Expect(f.FileType).To(Equal(expectedType))
		}
		Expect(files[0].IsOlderThan(files[1])).To(BeTrue())
	})
})
