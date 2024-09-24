package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"syscall"
)

func main() {
	flags := map[string]bool{
		"a": false,
		"r": false,
		"t": false,
		"R": false,
		"l": false,
	}
	var filestore []string
	// Parse command-line arguments for flags and target paths.
	for _, arg := range os.Args[1:] {
		if arg[0] == '-' {
			for _, flag := range arg[1:] {
				switch flag {
				case 'a':
					flags["a"] = true
				case 'r':
					flags["r"] = true
				case 't':
					flags["t"] = true
				case 'R':
					flags["R"] = true
				case 'l':
					flags["l"] = true
				}
			}
		} else {
			filestore = append(filestore, arg)
		}
	}

	if len(filestore) == 0 { // Default to current directory if no path is provided.
		filestore = append(filestore, ".")
	}
	if flags["l"] {
		longList(filestore)
	} else {
		shortList(filestore, flags)
	}
}

func check(file string) (bool, fs.FileInfo) {
	info, err := os.Stat(file)
	return err == nil, info
}

func shortList(filestore []string, flags map[string]bool) {
	var message []string
	var validFiles []string
	var directories []string
	var errorMessage string
	var files []string
	if flags["t"] {
		files = sortFilesByModTime(filestore)
	} else if flags["r"] {
		files = SortStringsDescending(filestore)
	} else {
		files = filestore
	}
	for _, file := range files {
		exist, fileInfo := check(file)
		if !exist {
			errorMessage = fmt.Sprintf("ls: cannot access '%v': No such file or directory", file)
			message = append(message, errorMessage)
			continue
		}
		if !fileInfo.IsDir() {
			if flags["a"] || file[0] != '.' {
				validFiles = append(validFiles, file)
			}
		} else {
			if flags["R"] {
				listRecursive(file, flags, "")
			} else {
				dirContents := directoryList([]string{}, file)
				if flags["t"] {
					dirContents = sortFilesByModTime(dirContents)
				} else if flags["r"] {
					dirContents = SortStringsDescending(dirContents)
				}
				for _, entry := range dirContents {
					if flags["a"] || entry[0] != '.' {
						directories = append(directories, entry)
					}
				}
				if flags["a"] {
					directories = append([]string{".", ".."}, directories...)
					for _, entry := range dirContents {
						if entry[0] == '.' && entry != "." && entry != ".." {
							directories = append(directories, entry)
						}
					}
				}
			}
		}
	}
	print(message)
	if len(validFiles) > 0 {
		printShort(validFiles)
	}
	if len(validFiles) > 0 && len(directories) > 0 && !flags["R"] {
		fmt.Println()
	}
	if !flags["R"] {
		printShort(directories)
	}
}

func directoryList(dircontent []string, file string) []string {
	content, err := os.Open(file)
	if err != nil {
		return dircontent
	}

	names, err := content.Readdirnames(0)
	if err != nil {
		return dircontent
	}
	return names
}

func directoryLongList(dircontent []string, file string) []string {
	var longList []string
	var format string
	content, err := os.Open(file)
	if err != nil {
		return dircontent
	}

	names, err := content.Readdir(0)
	if err != nil {
		return dircontent
	}
	for _, fileInfo := range names {
		size := fileInfo.Size()
		permission := fileInfo.Mode()
		name := fileInfo.Name()
		time := fileInfo.ModTime().Format("Jan 2 15:04") // Mon Jan 2 15:04:05 -0700 MST 2006
		user := userGroup()
		hardlinks := fileInfo.Sys().(*syscall.Stat_t).Nlink
		format = fmt.Sprintf("%v %d %v %v %v %s", permission, hardlinks, user, size, time, name)
		longList = append(longList, format)
	}
	return longList
}

func longList(files []string) {
	var message []string
	var validFiles []string
	var directories []string
	var errorMessage string

	for _, file := range files {
		exist, fileInfo := check(file)
		if !exist {
			errorMessage = fmt.Sprintf("ls: cannot access '%v': No such file or directory", file)
			message = append(message, errorMessage)
			continue
		}
		if !fileInfo.IsDir() {
			// drwxrwxr-x  5 hezron hezron  4096 Aug 14 17:10  python
			size := fileInfo.Size()
			permission := fileInfo.Mode()
			name := fileInfo.Name()
			time := fileInfo.ModTime().Format("Jan 2 15:04") // Mon Jan 2 15:04:05 -0700 MST 2006
			user := userGroup()
			hardlinks := fileInfo.Sys().(*syscall.Stat_t).Nlink
			format := fmt.Sprintf("%v %d %v %v %v %s", permission, hardlinks, user, size, time, name)
			validFiles = append(validFiles, format)
			continue

		}
		directories = directoryLongList(files, file)
	}
	print(message)
	print(validFiles)
	print(directories)
}

func listRecursive(path string, flags map[string]bool, indent string) {
	files, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", path, err)
		return
	}

	fmt.Printf("%s%s:\n", indent, path)
	var entries []string
	for _, file := range files {
		if flags["a"] || file.Name()[0] != '.' {
			entries = append(entries, file.Name())
		}
	}

	if flags["t"] {
		entries = sortFilesByModTime(entries)
	} else if flags["r"] {
		entries = SortStringsDescending(entries)
	} else {
		sort.Strings(entries)
	}

	printShort(entries)
	fmt.Println()

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.IsDir() {
			listRecursive(fullPath, flags, indent+"  ")
		}
	}
}

func userGroup() string {
	var userformat string
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	username := currentUser.Username
	usergroup, err := user.LookupGroupId(currentUser.Gid)
	if err != nil {
		panic(err)
	}
	userformat = fmt.Sprintf("%v %s", username, usergroup.Name)
	return userformat
}

func SortStringsDescending(slice []string) []string {
	n := len(slice)
	// Bubble sort algorithm
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			// Compare adjacent elements
			if slice[j] < slice[j+1] {
				// Swap if they are in the wrong order
				slice[j], slice[j+1] = slice[j+1], slice[j]
			}
		}
	}
	return slice
}

func sortFilesByModTime(files []string) []string {
	for i := 0; i < len(files)-1; i++ {
		for j := 0; j < len(files)-i-1; j++ {
			infoI, errI := os.Stat(files[j])
			infoJ, errJ := os.Stat(files[j+1])
			if errI != nil || errJ != nil {
				continue
			}
			if infoI.ModTime().Before(infoJ.ModTime()) {
				files[j], files[j+1] = files[j+1], files[j]
			}
		}
	}
	return files
}

func print(files []string) {
	for _, value := range files {
		fmt.Println(value)
	}
}

func printShort(files []string) {
	var result string
	for i, value := range files {
		result += value
		if i < len(files) {
			result += " "
		}
	}

	fmt.Println(result)
}
