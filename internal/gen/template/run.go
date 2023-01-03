package template

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hysios/mx/gen"
)

func runProtogen(ctx gen.Context) error {
	if err := runCmd(ctx, "buf", "mod", "update"); err != nil {
		return err
	}

	if err := runCmd(ctx, "buf", "format", "-w"); err != nil {
		return err
	}

	if err := runCmd(ctx, "buf", "generate"); err != nil {
		return err
	}

	return nil
}

func runModtidy(ctx gen.Context) error {
	return runCmd(ctx, "go", "mod", "tidy")
}

func runImports(ctx gen.Context) error {
	_ = runCmd(ctx, "goimports", "-w", ".")
	return nil
}

func runCmd(ctx gen.Context, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	outputDir := ctx.Value("OutputDir").(string)
	absdir, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}

	cmd.Dir = absdir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
