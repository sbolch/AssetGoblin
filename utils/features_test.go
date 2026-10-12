package utils

import (
	"github.com/regen-network/gocuke"
	"os"
	"testing"
)

func TestResourceFeatures(t *testing.T) {
	gocuke.NewRunner(t, &resourceSuite{}).Path("../features/utils/*.feature").After((*resourceSuite).After).Step(`^an open temporary file$`, (*resourceSuite).Open).Step(`^it is closed through the "([^"]*)" helper$`, (*resourceSuite).Close).Step(`^writing to the handle should fail$`, (*resourceSuite).WriteFails).Run()
}

type resourceSuite struct {
	gocuke.TestingT
	file *os.File
}

func (s *resourceSuite) Open() {
	s.Helper()
	var err error
	s.file, err = os.CreateTemp("", "assetgoblin-close-feature-")
	if err != nil {
		s.Fatal(err)
	}
}
func (s *resourceSuite) Close(helper string) {
	if helper == "file" {
		CloseFile(s.file)
	} else {
		CloseReader(s.file)
	}
}
func (s *resourceSuite) WriteFails() {
	s.Helper()
	if _, err := s.file.Write([]byte("test")); err == nil {
		s.Fatal("closed handle still writable")
	}
}
func (s *resourceSuite) After() {
	if s.file != nil {
		s.file.Close()
		os.Remove(s.file.Name())
		s.file = nil
	}
}
