package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

func main() {
	// To handle keyboard input better.
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	// Colors used across the program.
	redBackground := color.New(color.BgRed).SprintFunc()
	redText := color.New(color.FgRed).SprintFunc()

	// Goodies.
	var input string
	var scriptsDownloadPath string
	var schedulerSelected string
	var organizationSelected string
	var clusterCount int
	var clusterName string
	var submissionType string
	var numberOfWorkers int
	var hasSharedFileSystem bool
	var clusterMatlabRoot string
	var clusterHostname string
	var remoteJobStorageLocation string
	var downloadScriptsOnLanuch bool = true
	var caseNumber int

	// # Add some code that'll load any preferences for the program.
	// Setup for better Ctrl+C messaging. This is a channel to receive OS signals.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start a Goroutine to listen for signals.
	go func() {

		// Wait for the signal.
		<-signalChan

		// Handle the signal by exiting the program.
		fmt.Println(redBackground("\nExiting from user input..."))
		os.Exit(0)
	}()

	// Determine your OS.
	switch userOS := runtime.GOOS; userOS {
	case "darwin":
		scriptsDownloadPath = "/tmp"
	case "windows":
		scriptsDownloadPath = os.Getenv("TMP")
	case "linux":
		scriptsDownloadPath = "/tmp"
	default:
		scriptsDownloadPath = "unknown"
		fmt.Println(redText("\nYour operating system is unrecognized. Exiting."))
		os.Exit(0)
	}

	// Determine any user-defined settings.
	currentDir, err := os.Getwd() // Get the current working directory.
	if err != nil {
		fmt.Print(redText("\nError getting current working directory while looking for user settings : ", err, " Default settings will be used instead."))
		return
	} else {
		settingsPath := filepath.Join(currentDir, "settings.txt")

		// Check if the settings file exists.
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			// No settings found.
			return
		} else if err != nil {
			fmt.Print("\nError checking for user settings: ", err, " Default settings will be used instead.")
		} else {
			fmt.Print("\nCustom settings found!")
			file, err := os.Open(settingsPath)
			if err != nil {
				fmt.Println("\nError opening settings file: ", err, " Default settings will be used instead.")
				return
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				line := scanner.Text()

				if !strings.HasPrefix(line, "#") {
					// Process uncommented line.
					fmt.Println("\nProcessing line:", line)
					if strings.HasPrefix(line, "downloadScriptsOnLaunch") {
						fmt.Println("\nHey there!")
					}
				}
			}

			if err := scanner.Err(); err != nil {
				fmt.Println("Error reading settings file:", err)
			}
		}
	}

	if downloadScriptsOnLanuch {
		fmt.Println("Beginning download of integration scripts. Please wait.")

		var integrationScriptsURLs = map[string]string{
			"https://codeload.github.com/mathworks/matlab-parallel-slurm-plugin/zip/refs/heads/main":      "slurm.zip",
			"https://codeload.github.com/mathworks/matlab-parallel-pbs-plugin/zip/refs/heads/main":        "pbs.zip",
			"https://codeload.github.com/mathworks/matlab-parallel-lsf-plugin/zip/refs/heads/main":        "lsf.zip",
			"https://codeload.github.com/mathworks/matlab-parallel-htcondor-plugin/zip/refs/heads/main":   "htcondor.zip",
			"https://codeload.github.com/mathworks/matlab-parallel-gridengine-plugin/zip/refs/heads/main": "gridengine.zip",
			"https://codeload.github.com/mathworks/matlab-parallel-awsbatch-plugin/zip/refs/heads/main":   "awsbatch.zip",
			"https://codeload.github.com/mathworks/matlab-parallel-kubernetes-plugin/zip/refs/heads/main": "kubernetes.zip",
		}

		for url, zipArchive := range integrationScriptsURLs {
			zipArchivePath := filepath.Join(scriptsDownloadPath, zipArchive)
			err := downloadFile(url, zipArchivePath)
			if err != nil {
				fmt.Println("Failed to download integration scripts: ", err)
				continue
			}

			// Extract ZIP archives.
			schedulerName := strings.TrimSuffix(zipArchive, ".zip")
			unzipPath := filepath.Join(scriptsDownloadPath, schedulerName)

			// Check if the integration scripts directory already exists. Delete it if it is.
			if _, err := os.Stat(unzipPath); err == nil {

				err := os.RemoveAll(unzipPath)
				if err != nil {
					fmt.Println(redText("Failed to delete the existing integration scripts directory:", err))
					continue
				}
			}

			err = unzipFile(zipArchivePath, scriptsDownloadPath)
			if err != nil {
				fmt.Println(redText("Failed to extract integration scripts:", err))
				continue
			}

			if strings.Contains(zipArchivePath, "kubernetes.zip") {
				fmt.Println("Latest integration scripts downloaded and extracted successfully!")
			}
		}
	} else {
		fmt.Print("Integration scripts download skipped per user's settings.")
	}

	for {
		fmt.Print("Enter the organization's name.\n")
		organizationSelected, err = rl.Readline()
		if err != nil { // Handle error. For example, if user enters Ctrl+C.
			fmt.Println("Error reading line:", err)
			return
		}
		organizationSelected = strings.TrimSpace(organizationSelected)

		if organizationSelected == "" {
			fmt.Print("Invalid entry. ")
			continue
		} else {
			break
		}
	}

	for {
		fmt.Print("Enter the Salesforce Case Number associated with these scripts.\n")
		input, err = rl.Readline()
		if err != nil {
			fmt.Println("Error reading line:", err)
			return
		}
		input = strings.TrimSpace(input)

		// Don't accept anything other than numbers.
		if _, err := strconv.Atoi(input); err == nil {
			caseNumber, _ = strconv.Atoi(input)
			break
			fmt.Print(caseNumber) // Oh Go, it's okay, I promise, we'll use this shit.
		} else {
			fmt.Print(redText("Invalid entry. "))
			continue
		}
	}

	for {
		fmt.Print("Enter the number of clusters you'd like to make scripts for. Entering nothing will select 1.\n")
		input, err = rl.Readline()
		if err != nil {
			fmt.Println("Error reading line:", err)
			return
		}
		input = strings.TrimSpace(input)

		if input == "" {
			clusterCount = 1
			break
		}

		// Don't accept anything other than numbers.
		if _, err := strconv.Atoi(input); err == nil {
			clusterCount, _ = strconv.Atoi(input)
			break
		} else {
			fmt.Print(redText("Invalid entry. "))
			continue
		}
	}

	// Loop cluster creation for as many times as you specified.
	for i := 1; i <= clusterCount; i++ {
		for {
			fmt.Print("Enter cluster #", i, "'s name.\n")
			clusterName, err = rl.Readline()
			if err != nil {
				fmt.Println("Error reading line:", err)
				return
			}
			clusterName = strings.TrimSpace(clusterName)

			if clusterName == "" {
				fmt.Print(redText("Invalid entry. "))
				continue
			} else {
				break
			}
		}

		// waaaahhhh it's too difficult to just say "while".
		for {
			fmt.Print("Select the scheduler you'd like to use by entering its corresponding number. Entering nothing will select Slurm.\n")
			fmt.Print("[1 Slurm] [2 PBS] [3 LSF] [4 Grid Engine] [5 HTCondor] [6 AWS] [7 Kubernetes]\n")
			schedulerSelected, err = rl.Readline()
			if err != nil {
				fmt.Println("Error reading line:", err)
				return
			}
			schedulerSelected = strings.TrimSpace(schedulerSelected)

			break
		}

		for {
			fmt.Print("Select the submissions types you'd like to include by entering its corresponding number. Entering nothing will select both.\n")
			fmt.Print("[1 Remote] [2 Cluster] [3 Both]\n")
			submissionType, err = rl.Readline()
			if err != nil {
				fmt.Println("Error reading line:", err)
				return
			}
			submissionType = strings.TrimSpace(strings.ToLower(submissionType))

			if submissionType == "" {
				submissionType = "both"
				break
			} else if submissionType == "1" || submissionType == "2" || submissionType == "3" {
				switch submissionType {
				case "1":
					submissionType = "remote"
				case "2":
					submissionType = "cluster"
				case "3":
					submissionType = "both"
				}
				break
			} else if submissionType == "cluster" || submissionType == "remote" || submissionType == "both" {
				break
			} else {
				fmt.Print(redText("Invalid entry. "))
				continue
			}
		}

		for {
			fmt.Print("Enter the number of workers available on the cluster's license. Entering nothing will select 100,000.\n")
			input, err = rl.Readline()
			if err != nil {
				fmt.Println("Error reading line:", err)
				return
			}
			input = strings.TrimSpace(input)

			if input == "" {
				numberOfWorkers = 100000
				break
				fmt.Print(numberOfWorkers) // Again, Go, shut up.
			}

			// Don't accept anything other than numbers.
			if _, err := strconv.Atoi(input); err == nil {
				numberOfWorkers, _ = strconv.Atoi(input)
				break
			} else {
				fmt.Print(redText("Invalid entry. "))
				continue
			}
		}
		if submissionType == "remote" || submissionType == "both" {

			for {
				fmt.Print("Does the client have a shared filesystem with the cluster? (y/n)\n")
				input, err = rl.Readline()
				if err != nil {
					fmt.Println("Error reading line:", err)
					return
				}
				input = strings.TrimSpace(strings.ToLower(input))

				if input == "y" || input == "yes" {
					hasSharedFileSystem = true
					break
					fmt.Print(hasSharedFileSystem) // Again, shut up, Go. It'll be used at some point, I promise.
				} else if input == "n" || input == "no" {
					hasSharedFileSystem = false
					break
				} else {
					fmt.Print(redText("Invalid entry. "))
					continue
				}
			}

			for {
				fmt.Print("What is the full filepath of MATLAB on the cluster? (ex: /usr/local/MATLAB/R2023b)\n")
				clusterMatlabRoot, err = rl.Readline()
				if err != nil {
					fmt.Println("Error reading line:", err)
					return
				}
				clusterMatlabRoot = strings.TrimSpace(clusterMatlabRoot)

				if strings.Contains(clusterMatlabRoot, "/") || strings.Contains(clusterMatlabRoot, "\\") {
					break
				} else {
					fmt.Print(redText("Invalid filepath. "))
					continue
				}
			}

			for {
				fmt.Print("What is the hostname, FQDN, or IP address used to SSH to the cluster?\n")
				clusterHostname, err = rl.Readline()
				if err != nil {
					fmt.Println("Error reading line:", err)
					return
				}
				clusterHostname = strings.TrimSpace(clusterHostname)

				if clusterHostname == "" {
					fmt.Print(redText("Invalid entry. "))
					continue
				} else {
					break
				}
			}

			for {
				fmt.Print("Where will remote job storage location be on the cluster? Entering nothing will select /home/$User/.matlab/generic_cluster_jobs/" + clusterName + "/$Host\n")
				remoteJobStorageLocation, err = rl.Readline()
				if err != nil {
					fmt.Println("Error reading line:", err)
					return
				}
				remoteJobStorageLocation = strings.TrimSpace(remoteJobStorageLocation)

				if strings.Contains(remoteJobStorageLocation, "/") || strings.Contains(clusterMatlabRoot, "\\") {
					break
				} else if remoteJobStorageLocation == "" {
					remoteJobStorageLocation = "/home/$USER/.matlab/generic_cluster_jobs/" + clusterName + "/$HOST"
					break
				} else {
					fmt.Print(redText("Invalid filepath. "))
					continue
				}
			}
		}
		fmt.Print("Creating integration scripts for cluster #", i, "...\n")
		fmt.Print("Finished!\n")
		fmt.Print("Submitting to GitLab...\n")
		fmt.Print("Finished!\n")
	}
}

// Function to download a file from a given URL and save it to the specified path.
func downloadFile(url string, filePath string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

// Function to unzip integration scripts.
func unzipFile(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)

		// Reconstruct the file path on Windows to ensure proper subdirectories are created.
		// Don't know why other OSes don't need this.
		if runtime.GOOS == "windows" {
			path = filepath.Join(dest, file.Name)
			path = filepath.FromSlash(path)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return err
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}
	return nil
}
