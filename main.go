package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Model struct {
	Name string `json:"name"`
}

type ChatApp struct {
	myApp           fyne.App
	myWindow        fyne.Window
	chatContainer   *fyne.Container
	inputField      *widget.Entry
	statusLabel     *widget.Label
	modelSelect     *widget.Select
	cancelFunc      context.CancelFunc
	sendButton      *widget.Button
	clearButton     *widget.Button
	saveButton      *widget.Button
	cancelButton    *widget.Button
	queryInProgress bool // Add this field
}

func main() {
	app := NewChatApp()
	app.SetupModelSelection() // Move this up to ensure the modelSelect is initialized
	app.SetupUI()
	app.Run()
}

func NewChatApp() *ChatApp {
	myApp := app.NewWithID("github.com.ashprao.ollamachat")
	myWindow := myApp.NewWindow("Ollama Chat Interface")

	return &ChatApp{
		myApp:         myApp,
		myWindow:      myWindow,
		chatContainer: container.NewVBox(),
		inputField:    widget.NewMultiLineEntry(),
		statusLabel:   widget.NewLabel(""),
		// Initialize modelSelect here to prevent nil reference
		modelSelect: widget.NewSelect([]string{}, nil), // Initialize an empty select just in case
	}
}
func (c *ChatApp) SetupUI() {
	c.inputField.SetPlaceHolder("Type your query here...")
	c.inputField.Wrapping = fyne.TextWrapWord
	c.inputField.OnChanged = c.onInputFieldChanged

	scrollContainer := container.NewScroll(c.chatContainer)
	scrollContainer.SetMinSize(fyne.NewSize(400, 300))

	c.initButtons()

	modelSelectContainer := container.NewGridWrap(
		fyne.NewSize(c.calculateModelSelectWidth()+50, c.modelSelect.MinSize().Height),
		c.modelSelect,
	)

	statusArea := container.NewBorder(nil, nil, nil, c.cancelButton, c.statusLabel)

	buttons := container.NewVBox(c.sendButton, c.clearButton, c.saveButton, c.quitButton())

	inputArea := container.NewBorder(nil, nil, nil, buttons, c.inputField)

	content := container.NewBorder(
		statusArea,
		container.NewVBox(container.NewHBox(modelSelectContainer, widget.NewLabel("Model to be used.")), inputArea),
		nil,
		nil,
		scrollContainer,
	)
	c.myWindow.Resize(fyne.NewSize(400, 600))
	c.myWindow.SetContent(content)
}

func (c *ChatApp) Run() {
	c.myWindow.ShowAndRun()
}

func fetchModels() ([]Model, error) {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []Model `json:"models"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Models, nil
}

func sendQueryToLLM(ctx context.Context, model, query string, updateStatus func(string), updateChat func(string, bool)) error {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": query,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/generate", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	newStream := true
	for {
		var llmResp map[string]interface{}
		if err := decoder.Decode(&llmResp); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		if responseText, ok := llmResp["response"].(string); ok {
			updateChat(responseText, newStream)
			newStream = false
		}
	}

	updateStatus("")
	return nil
}

func (c *ChatApp) SetupModelSelection() {
	models, err := fetchModels()
	if err != nil {
		dialog.ShowError(err, c.myWindow)
		return
	}

	modelNames := c.extractModelNames(models)
	c.modelSelect = widget.NewSelect(modelNames, c.onModelSelect)

	savedModel := c.myApp.Preferences().StringWithFallback("selectedModel", "llama3.2:latest")
	c.modelSelect.SetSelected(savedModel)
}

func (c *ChatApp) initButtons() {
	c.sendButton = widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), c.onSendButtonTapped)
	c.clearButton = widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), c.onClearButtonTapped)
	c.saveButton = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), c.onSaveButtonTapped)
	c.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), c.onCancelButtonTapped)
	c.cancelButton.Hide()
	c.disableUtilityButtons()
}

func (c *ChatApp) disableUtilityButtons() {
	c.sendButton.Disable()
	c.clearButton.Disable()
	c.saveButton.Disable()
}

func (c *ChatApp) extractModelNames(models []Model) []string {
	modelNames := []string{}
	for _, model := range models {
		modelNames = append(modelNames, model.Name)
	}
	return modelNames
}

func (c *ChatApp) calculateModelSelectWidth() float32 {
	if c.modelSelect == nil || len(c.modelSelect.Options) == 0 {
		// Return a default width if modelSelect is not ready
		return 100 // You can adjust the default width as needed
	}
	longestModel := ""
	for _, model := range c.modelSelect.Options {
		if len(model) > len(longestModel) {
			longestModel = model
		}
	}
	return canvas.NewText(longestModel, nil).MinSize().Width
}

func (c *ChatApp) addMessageCard(content string, isUserMessage bool) *widget.Card {
	var title string
	if isUserMessage {
		title = "You:"
	} else {
		title = "LLM:"
	}

	richText := widget.NewRichTextFromMarkdown(content)
	richText.Wrapping = fyne.TextWrapWord
	messageCard := widget.NewCard(title, "", richText)
	c.chatContainer.Add(messageCard)
	c.enableUtilityButtons()

	return messageCard
}

func (c *ChatApp) enableUtilityButtons() {
	c.clearButton.Enable()
	c.saveButton.Enable()
}

func (c *ChatApp) onInputFieldChanged(content string) {
	c.updateSendButtonState() // Update the button state based on input content and query state
}

func (c *ChatApp) onSendButtonTapped() {
	if c.queryInProgress {
		return // Prevent sending if a query is already in progress
	}

	query := c.inputField.Text
	selectedModel := c.modelSelect.Selected
	if query == "" {
		return
	}

	c.queryInProgress = true  // Set the flag when a query starts
	c.updateSendButtonState() // Update the button states

	c.addMessageCard(query, true)
	c.inputField.SetText("")

	c.showProcessingStatus()
	scrollContainer := container.NewScroll(c.chatContainer)
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelFunc = cancelFunc

	go c.sendMessageToLLM(ctx, selectedModel, query, scrollContainer)
}

func (c *ChatApp) sendMessageToLLM(ctx context.Context, selectedModel, query string, scrollContainer *container.Scroll) {
	var card *widget.Card
	llmResponse := ""
	err := sendQueryToLLM(ctx, selectedModel, query, c.updateStatus, func(chunk string, newStream bool) {
		if newStream {
			llmResponse = chunk
			card = c.addMessageCard(llmResponse, false)
		} else {
			llmResponse += chunk
			c.updateRichText(card, llmResponse)
		}

		c.myWindow.Content().Refresh()
		if scrollContainer.Offset.Y >= scrollContainer.Content.Size().Height-scrollContainer.Size().Height-50 {
			scrollContainer.ScrollToBottom()
		}
	})

	c.queryInProgress = false // Reset the flag once the query finishes or is canceled
	c.updateSendButtonState() // Update the button states

	c.handleLLMResponseError(err)
}

func (c *ChatApp) updateStatus(status string) {
	c.statusLabel.SetText(status)
	c.myWindow.Content().Refresh()
}

func (c *ChatApp) updateRichText(card *widget.Card, content string) {
	if richText, ok := card.Content.(*widget.RichText); ok {
		richText.ParseMarkdown(content)
	}
}

func (c *ChatApp) handleLLMResponseError(err error) {
	if err != nil && err.Error() != "context canceled" {
		c.addMessageCard("\n**Error:** "+err.Error(), false)
	}

	c.clearProcessingStatus()
}

func (c *ChatApp) showProcessingStatus() {
	c.statusLabel.SetText("Processing...")
	c.cancelButton.Show()
}

func (c *ChatApp) clearProcessingStatus() {
	c.statusLabel.SetText("")
	c.cancelButton.Hide()
	c.myWindow.Content().Refresh()
}

func (c *ChatApp) updateSendButtonState() {
	if c.queryInProgress || c.inputField.Text == "" {
		c.sendButton.Disable()
	} else {
		c.sendButton.Enable()
	}
}

func (c *ChatApp) onCancelButtonTapped() {
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.addMessageCard("\n\n**Request canceled**", false)
		c.clearProcessingStatus()
	}

	c.queryInProgress = false // Reset the flag on cancel
	c.updateSendButtonState() // Update the button states
}

func (c *ChatApp) onClearButtonTapped() {
	c.chatContainer.Objects = nil
	c.disableUtilityButtons()
	c.myWindow.Content().Refresh()
}

func (c *ChatApp) onSaveButtonTapped() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}

		filename := writer.URI().Name()
		if !strings.HasSuffix(filename, ".txt") {
			filename += ".txt"
		}

		for _, obj := range c.chatContainer.Objects {
			if card, ok := obj.(*widget.Card); ok {
				writer.Write([]byte(card.Subtitle + "\n"))
			}
		}

		writer.Close()
		c.saveButton.Disable()
	}, c.myWindow)
}

func (c *ChatApp) onModelSelect(selected string) {
	c.myApp.Preferences().SetString("selectedModel", selected)
}

func (c *ChatApp) quitButton() *widget.Button {
	return widget.NewButtonWithIcon("Quit", theme.LogoutIcon(), func() {
		c.myApp.Quit()
	})
}
