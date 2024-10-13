package main

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"io"
// 	"log"
// 	"net/http"
// 	"strings"

// 	"fyne.io/fyne/v2"
// 	"fyne.io/fyne/v2/app"
// 	"fyne.io/fyne/v2/canvas"
// 	"fyne.io/fyne/v2/container"
// 	"fyne.io/fyne/v2/dialog"
// 	"fyne.io/fyne/v2/theme"
// 	"fyne.io/fyne/v2/widget"
// )

// // Define the structure for model
// type Model struct {
// 	Name string `json:"name"`
// }

// type LLMResponse struct {
// 	Response string `json:"response"`
// }

// // Fetch available models
// func fetchModels() ([]Model, error) {
// 	url := "http://localhost:11434/api/tags"
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	var result struct {
// 		Models []Model `json:"models"`
// 	}
// 	err = json.NewDecoder(resp.Body).Decode(&result)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return result.Models, nil
// }

// // Function to send the query to the local LLM and handle streaming response with cancellation context
// func sendQueryToLLM(ctx context.Context, model, query string, updateStatus func(string), updateChat func(string)) error {
// 	url := "http://localhost:11434/api/generate"

// 	requestBody, err := json.Marshal(map[string]interface{}{
// 		"model":  model, // Use the selected model
// 		"prompt": query,
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	// Create HTTP request with context for cancellation
// 	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
// 	if err != nil {
// 		return err
// 	}
// 	req.Header.Set("Content-Type", "application/json")

// 	// Send the HTTP POST request to the LLM
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	decoder := json.NewDecoder(resp.Body)

// 	for {
// 		var llmResp map[string]interface{}
// 		if err := decoder.Decode(&llmResp); err == io.EOF {
// 			break
// 		} else if err != nil {
// 			return err
// 		}

// 		if responseText, ok := llmResp["response"].(string); ok {
// 			updateChat(responseText)
// 		}
// 	}

// 	updateStatus("") // Clear status after the response is fully streamed
// 	return nil
// }

// func main() {
// 	myApp := app.NewWithID("com.ashprao.ollama-chat-interface")
// 	myWindow := myApp.NewWindow("Ollama Chat Interface")

// 	// Refactor: Use a wrapped Label instead of custom Entry
// 	chatHistory := widget.NewLabel("")
// 	chatHistory.Wrapping = fyne.TextWrapWord // Enable text wrapping
// 	// chatHistory.SetText("Chat history will appear here...")

// 	// Create a canvas rectangle to set the background color similar to the input field's background
// 	chatHistoryBackground := canvas.NewRectangle(theme.InputBackgroundColor()) // Use input background color

// 	// Wrap the label inside a container with the background rectangle
// 	chatHistoryContainer := container.NewStack(
// 		chatHistoryBackground, // Background color rectangle
// 		chatHistory,           // Wrapped label
// 	)

// 	inputField := widget.NewMultiLineEntry()
// 	inputField.SetPlaceHolder("Type your query here...")
// 	inputField.Wrapping = fyne.TextWrapWord

// 	statusLabel := widget.NewLabel("")

// 	// Create buttons
// 	sendButton := widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), nil)
// 	clearButton := widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), nil)
// 	saveButton := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), nil)
// 	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), nil)
// 	quitButton := widget.NewButtonWithIcon("Quit", theme.LogoutIcon(), func() {
// 		myApp.Quit() // This will quit the application
// 	})

// 	// Disable buttons initially
// 	sendButton.Disable()
// 	clearButton.Disable()
// 	saveButton.Disable()
// 	cancelButton.Hide() // Hide the cancel button initially

// 	// Fetch models and populate dropdown
// 	models, err := fetchModels()
// 	if err != nil {
// 		dialog.ShowError(err, myWindow)
// 		return
// 	}

// 	// Extract model names
// 	modelNames := []string{}
// 	for _, model := range models {
// 		modelNames = append(modelNames, model.Name)
// 	}

// 	// Get default model or saved model from preferences
// 	preferences := myApp.Preferences()
// 	savedModel := preferences.StringWithFallback("selectedModel", "llama3.2:latest")

// 	// Create dropdown for model selection
// 	modelSelect := widget.NewSelect(modelNames, func(selected string) {
// 		// Save the selected model in preferences
// 		preferences.SetString("selectedModel", selected)
// 	})
// 	modelSelect.SetSelected(savedModel) // Set the previously selected model

// 	// Find the longest string in the model names to set the dropdown's width
// 	longestModel := ""
// 	for _, model := range modelNames {
// 		if len(model) > len(longestModel) {
// 			longestModel = model
// 		}
// 	}
// 	log.Println("Longest Model: ", longestModel)
// 	// Measure the size of the longest model name and set it as the minimum size for the dropdown
// 	textSize := canvas.NewText(longestModel, nil).MinSize()
// 	modelSelect.Resize(fyne.NewSize(textSize.Width+100, modelSelect.MinSize().Height))

// 	// InputField changes to enable/disable Send button
// 	inputField.OnChanged = func(content string) {
// 		if len(content) == 0 {
// 			sendButton.Disable()
// 		} else {
// 			sendButton.Enable()
// 		}
// 	}

// 	// Track chat history changes to manage button states
// 	updateChatHistory := func(content string) {
// 		if len(content) == 0 {
// 			clearButton.Disable()
// 			saveButton.Disable()
// 		} else {
// 			clearButton.Enable()
// 			saveButton.Enable()
// 		}
// 		chatHistory.SetText(chatHistory.Text + content)
// 	}

// 	var cancelFunc context.CancelFunc // To store the cancel function

// 	// Send button functionality
// 	sendButton.OnTapped = func() {
// 		query := inputField.Text
// 		selectedModel := modelSelect.Selected
// 		if query != "" {
// 			updateChatHistory("\nYou: " + query + "\n\n")
// 			inputField.SetText("")

// 			statusLabel.SetText("Processing...")
// 			cancelButton.Show() // Show the cancel button when processing
// 			myWindow.Content().Refresh()

// 			updateChatHistory("LLM: ")
// 			myWindow.Content().Refresh()

// 			scrollContainer := container.NewScroll(chatHistory)

// 			// Create a cancellable context
// 			var ctx context.Context
// 			ctx, cancelFunc = context.WithCancel(context.Background())

// 			go func() {
// 				err := sendQueryToLLM(ctx, selectedModel, query, func(status string) {
// 					statusLabel.SetText(status)
// 					myWindow.Content().Refresh()
// 				}, func(chunk string) {
// 					updateChatHistory(chunk)
// 					myWindow.Content().Refresh()

// 					if scrollContainer.Offset.Y >= scrollContainer.Content.Size().Height-scrollContainer.Size().Height-50 {
// 						scrollContainer.ScrollToBottom()
// 					}
// 				})

// 				if err != nil {
// 					if err.Error() != "context canceled" {
// 						updateChatHistory("\nError: " + err.Error())
// 					}
// 					statusLabel.SetText("")
// 					cancelButton.Hide() // Hide the cancel button
// 					myWindow.Content().Refresh()
// 				} else {
// 					updateChatHistory("\n")
// 					statusLabel.SetText("")
// 					cancelButton.Hide() // Hide the cancel button after completion
// 					myWindow.Content().Refresh()
// 				}
// 			}()
// 		}
// 	}

// 	// Cancel button functionality
// 	cancelButton.OnTapped = func() {
// 		if cancelFunc != nil {
// 			cancelFunc() // Cancel the ongoing request
// 			updateChatHistory("\n\nRequest canceled\n")
// 			cancelButton.Hide() // Hide the cancel button after canceling
// 			myWindow.Content().Refresh()
// 		}
// 	}

// 	// Clear button functionality
// 	clearButton.OnTapped = func() {
// 		chatHistory.SetText("")
// 		myWindow.Content().Refresh()
// 	}

// 	// Save button functionality
// 	saveButton.OnTapped = func() {
// 		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
// 			if err == nil && writer != nil {
// 				filename := writer.URI().Name()

// 				if !strings.HasSuffix(filename, ".txt") {
// 					filename += ".txt"
// 				}

// 				writer.Write([]byte(chatHistory.Text))
// 				writer.Close()
// 				// Disable save after saving
// 				saveButton.Disable()
// 			}
// 		}, myWindow)
// 	}

// 	// Place the cancel button next to the status label
// 	statusArea := container.NewBorder(nil, nil, nil, cancelButton, statusLabel)

// 	buttons := container.NewVBox(sendButton, clearButton, saveButton, quitButton)

// 	inputArea := container.NewBorder(nil, nil, nil, buttons, inputField)

// 	scrollContainer := container.NewScroll(chatHistoryContainer)
// 	scrollContainer.SetMinSize(fyne.NewSize(400, 300)) // Set a minimum size for the scroll container

// 	modelSelectContainer := container.NewGridWrap(fyne.NewSize(textSize.Width+50, modelSelect.MinSize().Height), modelSelect)

// 	// Final layout including the model select dropdown
// 	content := container.NewBorder(
// 		statusArea,
// 		container.NewVBox(container.NewHBox(modelSelectContainer, widget.NewLabel("Model to be used.")), inputArea),
// 		nil,
// 		nil,
// 		scrollContainer,
// 	)

// 	myWindow.Resize(fyne.NewSize(400, 600))
// 	myWindow.SetContent(content)
// 	myWindow.ShowAndRun()
// }
