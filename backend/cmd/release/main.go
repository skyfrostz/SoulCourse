package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type target struct {
	GOOS   string
	GOARCH string
}

func main() {
	backendDir, err := os.Getwd()
	if err != nil {
		exitf("获取当前目录失败：%v", err)
	}
	repoRoot := filepath.Dir(backendDir)
	frontendDir := filepath.Join(repoRoot, "frontend")
	releaseDir := filepath.Join(repoRoot, "release")
	embedDistDir := filepath.Join(backendDir, "internal", "http", "webdist", "dist")

	fmt.Println("[打包] 开始构建前端静态资源")
	run("pnpm", []string{"build"}, frontendDir)

	fmt.Println("[打包] 同步前端产物到 Go 内嵌目录")
	syncDir(filepath.Join(frontendDir, "dist"), embedDistDir)

	fmt.Println("[打包] 编译跨平台二进制文件")
	if err := os.RemoveAll(releaseDir); err != nil {
		exitf("清理发布目录失败：%v", err)
	}
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		exitf("创建发布目录失败：%v", err)
	}

	targets := []target{
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
	}

	for _, item := range targets {
		buildBinary(backendDir, releaseDir, item)
	}

	copyIfExists(filepath.Join(repoRoot, ".env.example"), filepath.Join(releaseDir, ".env.example"))
	copyIfExists(filepath.Join(repoRoot, "README.md"), filepath.Join(releaseDir, "README.md"))

	fmt.Printf("[完成] 发布产物已生成到 %s\n", releaseDir)
}

func buildBinary(backendDir string, releaseDir string, item target) {
	fileName := fmt.Sprintf("soulcourse-%s-%s", item.GOOS, item.GOARCH)
	if item.GOOS == "windows" {
		fileName += ".exe"
	}
	outputPath := filepath.Join(releaseDir, fileName)

	cmd := exec.Command("go", "build", "-trimpath", "-o", outputPath, ".")
	cmd.Dir = backendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+item.GOOS,
		"GOARCH="+item.GOARCH,
	)

	fmt.Printf("[编译] %s/%s -> %s\n", item.GOOS, item.GOARCH, outputPath)
	if err := cmd.Run(); err != nil {
		exitf("构建 %s/%s 失败：%v", item.GOOS, item.GOARCH, err)
	}
}

func run(name string, args []string, dir string) {
	commandName := resolveCommand(name)
	cmd := exec.Command(commandName, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		exitf("执行命令失败：%s %v: %v", name, args, err)
	}
}

func resolveCommand(name string) string {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath(name + ".cmd"); err == nil {
			return name + ".cmd"
		}
	}
	return name
}

func syncDir(sourceDir string, targetDir string) {
	if err := os.RemoveAll(targetDir); err != nil {
		exitf("清理内嵌目录失败：%v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		exitf("创建内嵌目录失败：%v", err)
	}
	if err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		targetPath := filepath.Join(targetDir, relative)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		return copyFile(path, targetPath)
	}); err != nil {
		exitf("同步前端产物失败：%v", err)
	}
}

func copyIfExists(source string, target string) {
	if _, err := os.Stat(source); err != nil {
		return
	}
	if err := copyFile(source, target); err != nil {
		exitf("复制文件失败：%v", err)
	}
}

func copyFile(source string, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	output, err := os.Create(target)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return nil
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[失败] "+format+"\n", args...)
	os.Exit(1)
}
