package ocr

import (
	"fmt"
	"os/exec"
)

type Tesseract struct {
	command string
}

func Default() (*Tesseract, error) {
	return newTesseract("tesseract")
}

func newTesseract(command string) (*Tesseract, error) {
	t := &Tesseract{command: command}
	err := t.test()
	if err != nil {
		return nil, fmt.Errorf("tesseract initialization failed, %w", err)
	}

	return t, nil
}

func (t *Tesseract) test() error {
	cmd := exec.Command(t.command, "--version")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("testing tesseract installation, %w", err)
	}

	return nil
}

func (t *Tesseract) Run(file string) (string, error) {
	cmd := exec.Command(t.command, file, "stdout", "quiet")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("running tesseract, %w", err)
	}

	return string(out), nil
}
