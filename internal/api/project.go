package api

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type ProjectBody struct {
	Project Project `json:"project"`
}

type Project struct {
	Name                  string `json:"name"`
	DisplayName           string `json:"display_name"`
	OwnerOrganizationName string `json:"owner_organization_name,omitempty"`
}

type ProjectResponse struct {
	Name     string    `json:"name"`
	Done     bool      `json:"done"`
	Response *Response `json:"response"`
}

type Response struct {
	Type          string    `json:"@type"`
	Name          string    `json:"name"`
	DisplayName   string    `json:"display_name"`
	Location      *Location `json:"location"`
	OwnerUsername string    `json:"owner_username"`
}

type Location struct {
	GitPath *GitPath `json:"git_path"`
}

type GitPath struct{}

func (client *APIClient) ListProjects(token string) ([]*Project, error) {
	objmap, err := APIRequest[map[string]json.RawMessage](&RequestConfig{
		Client:       client,
		Method:       "GET",
		Token:        token,
		PathSegments: []string{"v1", "projects"},
	})
	if err != nil {
		return nil, err
	}

	var projects []*Project
	// If the projects field is not present, it means there are no projects
	// so we return an empty list of projects and no error.
	if _, ok := (*objmap)["projects"]; !ok {
		return []*Project{}, nil
	}
	err = json.Unmarshal((*objmap)["projects"], &projects)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Filter out featured projects
	var filteredProjects []*Project
	for _, p := range projects {
		if p.OwnerOrganizationName == FeaturedProjectsOrganization {
			continue
		}

		p.Name, err = ConvertProjectNameFromAPI(p.Name)
		if err != nil {
			return nil, err
		}
		filteredProjects = append(filteredProjects, p)
	}

	return filteredProjects, nil
}

func (client *APIClient) CreateProject(name string, token string) (*Project, error) {
	projectBody := &ProjectBody{
		Project: Project{
			DisplayName: name,
		},
	}

	body, err := json.MarshalIndent(projectBody, "", "  ")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	projectResponse, err := APIRequest[ProjectResponse](&RequestConfig{
		Client:       client,
		Method:       "POST",
		Body:         body,
		Token:        token,
		PathSegments: []string{"v1", "projects"},
	})
	if err != nil {
		return nil, err
	}
	projectBody.Project.Name = projectResponse.Response.Name

	return &projectBody.Project, nil
}
