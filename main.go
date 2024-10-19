/*
 * Copyright (c) 2024. Consulteer InCyber AG <incyber@consulteer.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
 * documentation files (the “Software”), to deal in the Software without restriction, including without limitation the
 * rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies or substantial portions of
 * the Software.
 *
 * THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
 * WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS
 * OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
 * OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	dataMeter    int64
	goodRequests int
	badRequests  int
	client       = &http.Client{
		Timeout: 5 * time.Second,
	}
	linkRegex = regexp.MustCompile(`(?:href=\")(https?:\/\/[^\"]+)(?:\")`)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "webtraffic",
	Short: "A web traffic generator",
	Long:  `A CLI tool to generate web traffic for demo purposes.`,
	Run:   run,
}

// init initializes the cobra command and sets up the flags and configuration
func init() {
	cobra.OnInitialize(initConfig, initLogging)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file "+
		"(default is $PWD/.webtraffic.yaml followed by $HOME/.webtraffic.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose logging")
	rootCmd.PersistentFlags().Int("max-depth", 10, "maximum depth for recursive browsing")
	rootCmd.PersistentFlags().Int("min-depth", 3, "minimum depth for recursive browsing")
	rootCmd.PersistentFlags().Int("max-wait", 10, "maximum wait time between requests")
	rootCmd.PersistentFlags().Int("min-wait", 5, "minimum wait time between requests")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("max_depth", rootCmd.PersistentFlags().Lookup("max-depth"))
	viper.BindPFlag("min_depth", rootCmd.PersistentFlags().Lookup("min-depth"))
	viper.BindPFlag("max_wait", rootCmd.PersistentFlags().Lookup("max-wait"))
	viper.BindPFlag("min_wait", rootCmd.PersistentFlags().Lookup("min-wait"))
}

// initConfig reads in config file and ENV variables if set.
// It sets up the configuration for the application using Viper.
// Possible config file paths:
// 1. Working directory
// 2. Home directory
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		currentDir, err := os.Getwd()
		cobra.CheckErr(err)
		viper.AddConfigPath(currentDir)

		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)

		viper.SetConfigType("yaml")
		viper.SetConfigName(".webtraffic")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.WithFields(log.Fields{
			"config_file": viper.ConfigFileUsed(),
			"err":         err,
		}).Error("Failed to read config file")
	} else {
		log.Infof("Using config file: %s", viper.ConfigFileUsed())
	}
}

// initLogging configures the logrus logger based on the debug flag in the configuration.
// If debug is true, it sets the log level to Debug, otherwise it sets it to Info.
func initLogging() {
	log.SetFormatter(&log.TextFormatter{
		QuoteEmptyFields: true,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
	})
	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
		log.Debug("Verbose logging enabled.")
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

// run is the main function that is executed when the command is run.
// It starts the web traffic generation process and continues indefinitely.
func run(cmd *cobra.Command, args []string) {
	log.Info("This webtraffic command will now run indefinitely, use Ctrl+C to abort.")
	log.WithFields(log.Fields{
		"minDepth": viper.GetInt("min_depth"),
		"maxDepth": viper.GetInt("max_depth"),
		"minWait":  viper.GetInt("min_wait"),
		"maxWait":  viper.GetInt("max_wait"),
	}).Debug("Configuration")

	for {
		randomURL := viper.GetStringSlice("root_urls")[rand.Intn(len(viper.GetStringSlice("root_urls")))]
		depth := rand.Intn(viper.GetInt("max_depth")-viper.GetInt("min_depth")+1) + viper.GetInt("min_depth")

		log.Infof("Randomly selected %s as the Root URL for recursive browsing.", randomURL)

		recursiveBrowse(randomURL, depth)

		// TODO: make this sleep time configurable
		pauseBeforeBrowse := time.Duration(10) * time.Second
		log.Infof("Pausing %s before choosing another Root URL.", pauseBeforeBrowse)
		time.Sleep(pauseBeforeBrowse)
	}
}

// recursiveBrowse performs a recursive browsing operation starting from the given URL.
// It continues until the specified depth is reached.
// If an error occurs or no valid links are found, it adds the URL to the blacklist.
//
// Parameters:
//   - url: The URL to start browsing from
//   - depth: The current depth of recursion
func recursiveBrowse(url string, depth int) {
	log.WithFields(log.Fields{
		"url":   url,
		"depth": depth,
	}).Info("Recursively browsing")

	if depth == 0 {
		doRequest(url)
		return
	}

	content, err := doRequest(url)
	if err != nil {
		log.WithFields(log.Fields{
			"url":   url,
			"error": err,
		}).Warn("Stopping and blacklisting: page error")
		viper.Set("blacklist", append(viper.GetStringSlice("blacklist"), url))
		return
	}

	validLinks := getLinks(content)
	log.WithField("linkCount", len(validLinks)).Debug("Valid links found")

	if len(validLinks) == 0 {
		log.WithField("url", url).Warn("Stopping and blacklisting: no links")
		viper.Set("blacklist", append(viper.GetStringSlice("blacklist"), url))
		return
	}

	sleepTime := rand.Intn(viper.GetInt("max_wait")-viper.GetInt("min_wait")+1) + viper.GetInt("min_wait")
	log.WithField("sleepTime", sleepTime).Debug("Pausing")
	time.Sleep(time.Duration(sleepTime) * time.Second)

	recursiveBrowse(validLinks[rand.Intn(len(validLinks))], depth-1)
}

// doRequest performs an HTTP GET request to the specified URL.
// It logs the request details, updates request counters, and handles rate limiting.
//
// Parameters:
//   - url: The URL to send the request to
//
// Returns:
//   - []byte: Content of the response body
//   - error: Any error that occurred during the request
func doRequest(url string) ([]byte, error) {
	log.WithField("url", url).Debug("Requesting page...")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", viper.GetString("user_agent"))

	resp, err := client.Do(req)
	if err != nil {
		time.Sleep(30 * time.Second)
		return nil, err
	}
	defer resp.Body.Close() // Ensure the body is always closed

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return content, err
	}

	pageSize := int64(len(content))
	dataMeter += pageSize

	log.WithFields(log.Fields{
		"pageSize":  hrBytes(pageSize),
		"dataMeter": hrBytes(dataMeter),
	}).Debug("Page size and data meter")

	if resp.StatusCode != 200 {
		badRequests++
		log.WithField("status", resp.StatusCode).Warn("Non-200 response status")
		if resp.StatusCode == 429 {
			log.Warn("We're making requests too frequently... sleeping longer...")
			viper.Set("min_wait", viper.GetInt("min_wait")+10)
			viper.Set("max_wait", viper.GetInt("max_wait")+10)
		}
	} else {
		goodRequests++
	}

	log.WithFields(log.Fields{
		"goodRequests": goodRequests,
		"badRequests":  badRequests,
	}).Debug("Request counters")

	return content, nil
}

// getLinks extracts all valid links from the given HTTP response body.
// It uses a regular expression to find links and filters out blacklisted ones.
//
// Parameters:
//   - content: An []byte containing the HTTP response body
//
// Returns:
//   - []string: A slice of valid links found in the body
func getLinks(content []byte) []string {
	links := linkRegex.FindAllString(string(content), -1)
	validLinks := make([]string, 0)
	for _, link := range links {
		cleanLink := link[6 : len(link)-1] // Remove href=" and "
		if !isBlacklisted(cleanLink) {
			validLinks = append(validLinks, cleanLink)
		}
	}
	return validLinks
}

// isBlacklisted checks if a given link is in the blacklist.
//
// Parameters:
//   - link: The link to check
//
// Returns:
//   - bool: true if the link is blacklisted, false otherwise
func isBlacklisted(link string) bool {
	for _, blacklisted := range viper.GetStringSlice("blacklist") {
		if strings.Contains(link, blacklisted) {
			return true
		}
	}
	return false
}

// hrBytes converts a byte size to a human-readable string.
// It uses base-1000 units (KB, MB, GB, etc.) for conversion.
//
// Parameters:
//   - bytes: The number of bytes to convert
//
// Returns:
//   - string: A human-readable representation of the byte size
func hrBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// main is the entry point of the application.
// It executes the root command and handles any errors.
func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
