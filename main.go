package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pqkallio/nand2tetris-jack-compiler/compilationengine"
	"github.com/pqkallio/nand2tetris-jack-compiler/tokenizer"
	"github.com/pqkallio/nand2tetris-jack-compiler/vm"
)

const (
	file = iota
	dir
)

type fileInfo struct {
	fullPath string
	file     fs.FileInfo
}

type pathData struct {
	pathType int
	files    []fileInfo
}

func openDir(filePath string) (pathData, error) {
	data := pathData{pathType: dir}

	files := []fileInfo{}

	err := filepath.Walk(filePath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && path != filePath {
			return filepath.SkipDir
		}

		if strings.HasSuffix(info.Name(), ".jack") {
			files = append(files, fileInfo{path, info})
		}

		return nil
	})

	if err != nil {
		return data, err
	}

	data.files = files

	return data, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var data pathData

	args := os.Args[1:]

	if len(args) != 1 {
		log.Fatalf("please provide only the file or folder to compile")
	}

	fn := args[0]

	stat, err := os.Stat(fn)
	if err != nil {
		log.Fatalf("Unable to check path %s", fn)
	}

	switch mode := stat.Mode(); {
	case mode.IsDir():
		data, err = openDir(fn)
		if err != nil {
			log.Fatalf("Unable to read directory %s", fn)
		}
	case mode.IsRegular():
		data.pathType = file
		data.files = []fileInfo{{fn, stat}}
	}

	for _, f := range data.files {
		compileFile(&f)
	}
}

func compileFile(f *fileInfo) {
	log.Printf("compiling file %s", f.file.Name())
	in, err := os.Open(f.fullPath)
	if err != nil {
		log.Fatalf("error opening file %s: %s", f.fullPath, err)
	}

	defer in.Close()

	split := strings.Split(f.fullPath, ".jack")
	vmOutName := split[0] + ".vm"

	vmOut, err := os.Create(vmOutName)
	if err != nil {
		log.Fatalf("error opening file %s: %s", vmOutName, err)
	}

	vmWriter := vm.New(vmOut)

	t := tokenizer.New(in)
	c := compilationengine.New(t, vmWriter)

	err = c.Compile()
	if err != nil {
		log.Fatalf("compilation of file %s failed: %s", f.fullPath, err.Error())
	}
}
