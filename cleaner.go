package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"
)

const dockerHubAPI = "https://hub.docker.com/v2"

// Struct for image tag response
type Tag struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"last_updated"`
	FullSize    int64     `json:"full_size"` // size in bytes
}

type TagsResponse struct {
	Results []Tag `json:"results"`
	Next    string `json:"next"`
}

// Get all tags from the repository
func getTags(user, repo, token string) ([]Tag, error) {
	var tags []Tag
	urlStr := fmt.Sprintf("%s/repositories/%s/%s/tags?page_size=100", dockerHubAPI, user, repo)

	client := &http.Client{}
	for urlStr != "" {
		req, _ := http.NewRequest("GET", urlStr, nil)
		req.Header.Set("Authorization", "JWT "+token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to fetch tags: %s", resp.Status)
		}

		var tr TagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
			return nil, err
		}

		tags = append(tags, tr.Results...)
		urlStr = tr.Next
	}
	return tags, nil
}

// Delete tag
func deleteTag(user, repo, tag, token string) error {
	urlStr := fmt.Sprintf("%s/repositories/%s/%s/tags/%s/", dockerHubAPI, user, repo, tag)
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", urlStr, nil)
	req.Header.Set("Authorization", "JWT "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete %s: %s", tag, string(body))
	}
	return nil
}

// Get JWT token
func login(user, password string) (string, error) {
	data := url.Values{}
	data.Set("username", user)
	data.Set("password", password)

	resp, err := http.PostForm(dockerHubAPI+"/users/login/", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("login failed: %s", resp.Status)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result["token"], nil
}

// Compute the total volume occupied by all tags present in the repository
func sumSize(tags []Tag) int64 {
	var total int64
	for _, t := range tags {
		total += t.FullSize
	}
	return total
}

func main() {
	user := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")
	repo := os.Getenv("DOCKER_REPOSITORY")
	keepCountStr := os.Getenv("KEEP_COUNT")
	maxSizeStr := os.Getenv("MAX_SIZE_MB")

	if user == "" || password == "" || repo == "" {
		log.Fatal("Missing required environment variables: DOCKER_USERNAME, DOCKER_PASSWORD, DOCKER_REPOSITORY")
	}

	var keepCount int
	var err error
	if keepCountStr != "" {
		keepCount, err = strconv.Atoi(keepCountStr)
		if err != nil || keepCount < 0 {
			log.Fatalf("Invalid KEEP_COUNT value: %s", keepCountStr)
		}
	} else {
		keepCount = -1 // not enforced
	}

	var maxSizeMB int64 = -1
	if maxSizeStr != "" {
		maxSizeMB, err = strconv.ParseInt(maxSizeStr, 10, 64)
		if err != nil {
			log.Fatalf("Invalid MAX_SIZE_MB value: %s", maxSizeStr)
		}
	}

	// Authenticate
	token, err := login(user, password)
	if err != nil {
		log.Fatal("Login failed:", err)
	}

	// Fetch tags
	tags, err := getTags(user, repo, token)
	if err != nil {
		log.Fatal("Failed to get tags:", err)
	}

	// Sort by last updated (newest first)
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].LastUpdated.After(tags[j].LastUpdated)
	})

	// Apply KEEP_COUNT if set
	if keepCount >= 0 && len(tags) > keepCount {
		toDelete := tags[keepCount:]
		for _, t := range toDelete {
			fmt.Println("Deleting (exceeds count):", t.Name)
			if err := deleteTag(user, repo, t.Name, token); err != nil {
				log.Println("Error deleting:", t.Name, err)
			}
		}
		tags = tags[:keepCount]
	}

	// Apply MAX_SIZE_MB if set
	if maxSizeMB > 0 {
		for sumSize(tags) > maxSizeMB*1024*1024 && len(tags) > 0 {
			oldest := tags[len(tags)-1]
			fmt.Printf("Deleting (exceeds size, total=%.2fMB): %s\n", float64(sumSize(tags))/(1024*1024), oldest.Name)
			if err := deleteTag(user, repo, oldest.Name, token); err != nil {
				log.Println("Error deleting:", oldest.Name, err)
			}
			tags = tags[:len(tags)-1]
		}
	}

	fmt.Printf("Cleanup complete. Remaining images: %d, total size: %.2f MB\n",
		len(tags), float64(sumSize(tags))/(1024*1024))
}
