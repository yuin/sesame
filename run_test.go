package sesame_test

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestRun(t *testing.T) {
	executable := "sesame"
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", executable, "./cmd/sesame")
	defer os.Remove(executable)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err.Error() + ":" + string(out))
	}

	pwd, _ := os.Getwd()
	defer os.Chdir(pwd)
	os.Chdir("./testdata/testmod")
	cmd = exec.Command("../../" + executable)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err.Error() + ":" + string(out))
	}
	cmd = exec.Command("go", "test", "./...")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err.Error() + ":" + string(out))
	}
	t.Log(string(out))
}
