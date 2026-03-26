package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sdnxshu/cascoon/internal/queue"
	"gopkg.in/yaml.v3"
)

type Workflow struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
	Steps []Step `yaml:"steps"`
}

type Step struct {
	Name    string                 `yaml:"name"`
	Run     string                 `yaml:"run"`
	Env     map[string]string      `yaml:"env,omitempty"`
	Timeout string                 `yaml:"timeout,omitempty"`
	Extras  map[string]interface{} `yaml:",inline"`
}

func (w *Workflow) Validate() error {
	if w.Image == "" {
		return errors.New("image is required")
	}
	if len(w.Steps) == 0 {
		return errors.New("at least one step is required")
	}

	for i, step := range w.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i)
		}
		if step.Run == "" {
			return fmt.Errorf("step %d (%s): run is required", i, step.Name)
		}
	}

	return nil
}

func LoadWorkflow(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}

	if err := wf.Validate(); err != nil {
		return nil, err
	}

	return &wf, nil
}

func LoadWorkflows(dir string) ([]*Workflow, error) {
	var workflows []*Workflow

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Only process .yaml / .yml files
		if d.IsDir() || !(filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			return nil
		}

		wf, err := LoadWorkflow(path)
		if err != nil {
			return fmt.Errorf("error in %s: %w", path, err)
		}

		workflows = append(workflows, wf)
		return nil
	})

	return workflows, err
}

type RequestBody struct {
	Repo string `json:"repo"`
}

func RunHandler(c *gin.Context) {
	var body RequestBody

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	dir := ".jennings/workflows"

	workflows, err := LoadWorkflows(dir)
	if err != nil {
		panic(err)
	}

	for _, wf := range workflows {
		fmt.Println("=================================")
		fmt.Println("Name:", wf.Name)
		fmt.Println("Image:", wf.Image)

		fmt.Println("\nSteps:")
		for _, step := range wf.Steps {
			fmt.Printf("- %s: %s\n", step.Name, step.Run)

			if len(step.Env) > 0 {
				fmt.Println("  Env:", step.Env)
			}

			if step.Timeout != "" {
				fmt.Println("  Timeout:", step.Timeout)
			}

			if len(step.Extras) > 0 {
				fmt.Println("  Extras:", step.Extras)
			}
		}
	}

	client := queue.NewClient()
	defer client.Close()

	task := queue.NewTask(body.Repo)

	if _, err := client.Enqueue(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Queued successfully 🚀",
	})
}
