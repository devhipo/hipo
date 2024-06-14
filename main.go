package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var ArchitecturesMap = map[string]string{
	"amd64": "x64",
}

type AvailableReleaseResponse struct {
	AvailableLtsReleases     []uint `json:"available_lts_releases"`
	AvailableReleases        []uint `json:"available_releases"`
	MostRecentFeatureRelease uint   `json:"most_recent_feature_release"`
	MostRecentFeatureVersion uint   `json:"most_recent_feature_version"`
	MostRecentLts            uint   `json:"most_recent_lts"`
	TipVersion               uint   `json:"tip_version"`
}

func main() {

	if len(os.Args) == 1 {
		InitHipo()
	} else if len(os.Args) == 2 {
		destPath := downloadFile(os.Args[1])
		executeFile(destPath)
	}
}

func downloadFile(Args string) string {

	parts := strings.Split(Args, ":")
	if len(parts) != 3 {
		fmt.Println("Invalid coordinate format. Use <group:artifact:version>")
		return ""
	}

	// Go to link
	groupID := parts[0]
	artifactID := parts[1]
	version := parts[2]

	groupPath := strings.ReplaceAll(groupID, ".", "/")
	artifactFilename := fmt.Sprintf("%s-%s.jar", artifactID, version)
	url := fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s", groupPath, artifactID, version, artifactFilename)

	homeDir, err := os.UserHomeDir()

	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	// Create directory to save the JAR file
	destDir := filepath.Join(homeDir, ".hipo", groupPath, artifactID, version)
	err2 := os.MkdirAll(destDir, 0755)
	if err2 != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return ""
	}

	destPath := filepath.Join(destDir, artifactFilename)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("failed to make GET request: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("failed to download file: status code %d", resp.StatusCode)
		return ""
	}

	out, err := os.Create(destPath)
	if err != nil {
		fmt.Printf("failed to create file: %v", err)
		return ""
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Printf("failed to copy response body to file: %v", err)
		return ""
	}

	// fmt.Printf("Downloaded to %s\n", destPath)
	return destPath
}

func executeFile(jarFilePath string) {

	homeDir, err := os.UserHomeDir()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	jreParentDir := filepath.Join(homeDir, ".hipo", "jre")

	// List the directories in the parent JRE directory
	dirs, err := os.ReadDir(jreParentDir)
	if err != nil {
		fmt.Println("Error reading JRE directory:", err)
		return
	}

	if len(dirs) == 0 {
		fmt.Println("No JDK directories found")
		return
	}

	// Assume there is only one directory and get its name
	jdkDir := dirs[0].Name()

	// Path to the java executable
	javaExecPath := filepath.Join(homeDir, ".hipo", "jre", jdkDir, "bin", "java")

	cmd := exec.Command(javaExecPath, "-jar", jarFilePath)

	// Set the command's standard output and standard error
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Println("Error running the Java command:", err)
		return
	}

	//fmt.Println("Java application ran successfully")
}

func InitHipo() {
	hipoHome, done := PrepareHipoHome()

	if !done {
		return
	}

	mostRecentJavaRelease, done := GetLatestJavaRelease()

	if !done {
		return
	}

	var arch = ArchitecturesMap[runtime.GOARCH]
	var osName = runtime.GOOS

	done = DownloadJava(mostRecentJavaRelease, osName, arch, hipoHome)

	if !done {
		return
	}
}

func PrepareHipoHome() (string, bool) {

	homeDir, err := os.UserHomeDir()

	if err != nil {
		fmt.Println("Error:", err)
		return "", false
	}

	var hipoHomeDir = homeDir + "/.hipo"

	// Check if the directory already exists
	if _, err := os.Stat(hipoHomeDir); !os.IsNotExist(err) {
		// fmt.Println("Directory already exists:", hipoHomeDir)
		return "", false
	}

	err = os.MkdirAll(hipoHomeDir, 0755)

	if err != nil {
		fmt.Println("Error:", err)
		return "", false
	}

	return hipoHomeDir, true
}

func DownloadJava(release uint, osName string, arch string, hipoHome string) bool {

	url := fmt.Sprintf("https://api.adoptium.net/v3/binary/latest/%d/ga/%s/%s/jre/hotspot/normal/eclipse?project=jdk",
		release, osName, arch)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error: failed to make GET request:", err)
		return false
	}
	defer resp.Body.Close()

	// Check if the HTTP response status is OK
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: HTTP request failed with status code %d\n", resp.StatusCode)
		return false
	}

	var hipoJreDir = hipoHome + "/jre"

	err = os.MkdirAll(hipoJreDir, 0755)
	if err != nil {
		fmt.Println("Error: failed to create directory:", err)
		return false
	}

	// Create a temporary file to store the downloaded Java runtime
	tmpFile, err := os.CreateTemp("", "jre-*.zip")
	if err != nil {
		fmt.Println("Error: failed to create temporary file:", err)
		return false
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temporary file
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body) // Save binary to temporary file
	if err != nil {
		fmt.Println("Error: failed to copy response body to file:", err)
		return false
	}

	// Seek to the beginning of the temporary file to read it
	if _, err := tmpFile.Seek(0, 0); err != nil {
		fmt.Println("Error: failed to seek to beginning of file:", err)
		return false
	}

	// Extract the zip file contents to the destination directory
	if err := ExtractZip(tmpFile.Name(), hipoHome+"/jre"); err != nil {
		fmt.Println("Error extracting ZIP file:", err)
		return false
	}

	return true
}

func ExtractZip(src, destination string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destination, f.Name)

		// For ZipSlip error
		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func GetLatestJavaRelease() (uint, bool) {
	resp, err := http.Get("https://api.adoptium.net/v3/info/available_releases")

	if err != nil {
		fmt.Println("Error:", err)
		return 0, false
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("Error:", err)
		return 0, false
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error:", err)
		}
	}(resp.Body)

	var response AvailableReleaseResponse

	err = json.Unmarshal(body, &response)

	if err != nil {
		fmt.Println("Error:", err)
		return 0, false
	}

	return response.MostRecentFeatureRelease, true
}
