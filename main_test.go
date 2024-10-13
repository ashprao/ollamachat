package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"fyne.io/fyne/v2/widget"
)

// Mock HTTP client for testing
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func Test_fetchModels(t *testing.T) {
	tests := []struct {
		name       string
		mockClient *mockHTTPClient
		want       []Model
		wantErr    bool
	}{
		{
			name: "Successful fetch",
			mockClient: &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					models := []Model{{Name: "Model1"}, {Name: "Model2"}}
					body, _ := json.Marshal(struct {
						Models []Model `json:"models"`
					}{Models: models})
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}, nil
				},
			},
			want:    []Model{{Name: "Model1"}, {Name: "Model2"}},
			wantErr: false,
		},
		{
			name: "Network error",
			mockClient: &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "JSON decoding error",
			mockClient: &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("invalid json")),
					}, nil
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: tt.mockClient,
			}
			got, err := fetchModels(client)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchModels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchModels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sendQueryToLLM(t *testing.T) {
	tests := []struct {
		name          string
		mockClient    *mockHTTPClient
		model         string
		query         string
		wantErr       bool
		expectedChat  string
		expectedError string
	}{
		{
			name: "Successful query",
			mockClient: &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					response := map[string]interface{}{
						"response": "LLM response",
					}
					body, _ := json.Marshal(response)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}, nil
				},
			},
			model:        "testModel",
			query:        "testQuery",
			wantErr:      false,
			expectedChat: "LLM response",
		},
		{
			name: "Network error",
			mockClient: &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network error")
				},
			},
			model:         "testModel",
			query:         "testQuery",
			wantErr:       true,
			expectedError: "network error",
		},
		{
			name: "Invalid JSON response",
			mockClient: &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("invalid json")),
					}, nil
				},
			},
			model:         "testModel",
			query:         "testQuery",
			wantErr:       true,
			expectedError: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chatOutput string

			err := sendQueryToLLM(context.Background(), tt.model, tt.query, func(status string) {}, func(chat string, newStream bool) {
				chatOutput = chat
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("sendQueryToLLM() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("sendQueryToLLM() error = %v, expectedError %v", err, tt.expectedError)
			}

			if chatOutput != tt.expectedChat {
				t.Errorf("sendQueryToLLM() chatOutput = %v, expectedChat %v", chatOutput, tt.expectedChat)
			}
		})
	}
}

func TestChatApp_SetupModelSelection(t *testing.T) {
	app := NewChatApp()
	app.SetupModelSelection()

	if len(app.modelSelect.Options) != 2 {
		t.Errorf("SetupModelSelection() modelSelect.Options = %v, want 2 options", len(app.modelSelect.Options))
	}
}

func TestChatApp_initButtons(t *testing.T) {
	app := NewChatApp()
	app.initButtons()

	if app.sendButton == nil || app.clearButton == nil || app.saveButton == nil || app.cancelButton == nil {
		t.Error("initButtons() did not initialize all buttons")
	}
}

func TestChatApp_extractModelNames(t *testing.T) {
	app := NewChatApp()
	models := []Model{{Name: "Model1"}, {Name: "Model2"}}
	modelNames := app.extractModelNames(models)

	expected := []string{"Model1", "Model2"}
	if !reflect.DeepEqual(modelNames, expected) {
		t.Errorf("extractModelNames() = %v, want %v", modelNames, expected)
	}
}

func TestChatApp_calculateModelSelectWidth(t *testing.T) {
	app := NewChatApp()
	app.modelSelect = widget.NewSelect([]string{"Short", "LongerModelName"}, nil)

	width := app.calculateModelSelectWidth()
	if width <= 0 {
		t.Errorf("calculateModelSelectWidth() = %v, want > 0", width)
	}
}
