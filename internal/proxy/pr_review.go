package proxy

import (
	"encoding/json"
	"fmt"
	"time"
)

// PRReviewRequest identifies either a PR or an immutable pair of file versions.
type PRReviewRequest struct {
	Action    string `json:"action"`
	Reference string `json:"reference,omitempty"`
	URL       string `json:"url,omitempty"`
	Base      string `json:"base,omitempty"`
	Head      string `json:"head,omitempty"`
	Path      string `json:"path,omitempty"`
	OldPath   string `json:"old_path,omitempty"`
	Status    string `json:"status,omitempty"`
}

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Body   string `json:"body"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

type PRFile struct {
	Path      string `json:"filename"`
	OldPath   string `json:"previous_filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type PRReviewResult struct {
	Success        bool          `json:"success"`
	Error          string        `json:"error"`
	Requested      []PullRequest `json:"requested"`
	Assigned       []PullRequest `json:"assigned"`
	PR             PullRequest   `json:"pr"`
	Base           string        `json:"base"`
	Head           string        `json:"head"`
	Files          []PRFile      `json:"files"`
	Left           string        `json:"left"`
	Right          string        `json:"right"`
	Binary         bool          `json:"binary"`
	NavigationPath string        `json:"navigation_path"`
}

func (pm *ProxyManager) ReviewPR(workingDir string, query PRReviewRequest) (*PRReviewResult, error) {
	client, actualPath, err := pm.getClient(workingDir)
	if err != nil {
		return nil, err
	}
	response, err := client.sendRequestWithDuration(JSONRPCRequest{
		Method: "review_pr",
		Params: map[string]interface{}{"working_dir": actualPath, "query": query},
		ID:     client.getNextRequestID(),
	}, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	if msg, ok := response["error"]; ok {
		return nil, fmt.Errorf("PR review: %v", msg)
	}
	data, err := json.Marshal(response["result"])
	if err != nil {
		return nil, err
	}
	var result PRReviewResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("PR review: %s", result.Error)
	}
	if host, _, remote := parseRemotePath(workingDir); remote && result.NavigationPath != "" {
		result.NavigationPath = host + ":" + result.NavigationPath
	}
	return &result, nil
}

type PRCheckoutFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Error   string `json:"error"`
}

type PRCheckoutResult struct {
	Success bool             `json:"success"`
	Error   string           `json:"error"`
	Root    string           `json:"root"`
	Files   []PRCheckoutFile `json:"files"`
}

func (pm *ProxyManager) CheckoutPR(dir, reference, head string, openFiles []string) (*PRCheckoutResult, error) {
	client, actualPath, err := pm.getClient(dir)
	if err != nil {
		return nil, err
	}
	host, _, remote := parseRemotePath(dir)
	var paths []string
	for _, path := range openFiles {
		fileHost, actualFile, fileRemote := parseRemotePath(path)
		if remote == fileRemote && host == fileHost {
			paths = append(paths, actualFile)
		}
	}
	if paths == nil {
		paths = []string{}
	}
	response, err := client.sendRequestWithDuration(JSONRPCRequest{
		Method: "checkout_pr", ID: client.getNextRequestID(),
		Params: map[string]interface{}{"working_dir": actualPath, "reference": reference, "head": head, "open_files": paths},
	}, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	if msg, ok := response["error"]; ok {
		return nil, fmt.Errorf("PR checkout: %v", msg)
	}
	data, err := json.Marshal(response["result"])
	if err != nil {
		return nil, err
	}
	var result PRCheckoutResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if remote {
		if result.Root != "" {
			result.Root = host + ":" + result.Root
		}
		for i := range result.Files {
			result.Files[i].Path = host + ":" + result.Files[i].Path
		}
	}
	if !result.Success {
		return &result, fmt.Errorf("PR checkout: %s", result.Error)
	}
	return &result, nil
}
