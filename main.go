package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Model struct {
	Name string `json:"name"`
}

type ChatMessage struct {
	Sender    string `json:"sender"` // "user" or "llm"
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"` // Optional, but useful
}
type ChatApp struct {
	myApp           fyne.App
	myWindow        fyne.Window
	chatContainer   *fyne.Container
	scrollContainer *container.Scroll
	inputField      *widget.Entry
	statusLabel     *widget.Label
	modelSelect     *widget.Select
	cancelFunc      context.CancelFunc
	sendButton      *widget.Button
	clearButton     *widget.Button
	saveButton      *widget.Button
	cancelButton    *widget.Button
	queryInProgress bool
	messages        []ChatMessage // Store chat messages
}

func main() {
	app := NewChatApp()
	app.SetupUI()
	app.Run()
}

func NewChatApp() *ChatApp {
	myApp := app.NewWithID("github.com.ashprao.ollamachat")
	myWindow := myApp.NewWindow("Ollama Chat Interface")
	myWindow.Resize(fyne.NewSize(600, 700))

	chatApp := &ChatApp{
		myApp:         myApp,
		myWindow:      myWindow,
		chatContainer: container.NewVBox(),
		inputField:    widget.NewMultiLineEntry(),
		statusLabel:   widget.NewLabel(""),
		// Initialize modelSelect here to prevent nil reference
		modelSelect: widget.NewSelect([]string{}, nil), // Initialize an empty select just in case
	}

	return chatApp
}

func getHistoryFileURI() (fyne.URI, error) {
	root := fyne.CurrentApp().Storage().RootURI()
	uri, err := storage.Child(root, "chat_history.json")
	return uri, err
}

func (c *ChatApp) SaveChatHistory() error {
	uri, err := getHistoryFileURI()
	if err != nil {
		return err
	}

	file, err := storage.Writer(uri)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(c.messages)
	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}

func (c *ChatApp) LoadChatHistory() error {
	uri, err := getHistoryFileURI()
	if err != nil {
		return err
	}

	file, err := storage.Reader(uri)
	if err != nil {
		// If the file does not exist, treat as empty history and do not show an error
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file") {
			c.messages = []ChatMessage{}
			return nil
		}
		return err
	}
	defer file.Close()

	var messages []ChatMessage
	if err := json.NewDecoder(file).Decode(&messages); err != nil {
		return err
	}

	c.messages = messages
	return nil
}

func (c *ChatApp) SetupUI() {
	c.inputField.SetPlaceHolder("Type your query here...")
	c.inputField.Wrapping = fyne.TextWrapWord
	c.inputField.OnChanged = c.onInputFieldChanged

	c.initButtons()

	modelSelectContainer := container.NewGridWrap(
		fyne.NewSize(c.calculateModelSelectWidth()+50, c.modelSelect.MinSize().Height),
		c.modelSelect,
	)

	statusArea := container.NewBorder(nil, nil, nil, c.cancelButton, c.statusLabel)
	buttons := container.NewVBox(c.sendButton, c.clearButton, c.saveButton, c.quitButton())
	inputArea := container.NewBorder(nil, nil, nil, buttons, c.inputField)

	c.scrollContainer = container.NewScroll(c.chatContainer)
	c.scrollContainer.SetMinSize(fyne.NewSize(400, 300))

	for _, msg := range c.messages {
		isUserMessage := msg.Sender == "user"
		c.addMessageCard(msg.Content, isUserMessage, false)
	}

	content := container.NewBorder(
		statusArea,
		container.NewVBox(container.NewHBox(modelSelectContainer, widget.NewLabel("Model to be used.")), inputArea),
		nil,
		nil,
		c.scrollContainer,
	)
	// c.myWindow.Resize(fyne.NewSize(600, 700))
	c.myWindow.SetContent(content)
}

func (c *ChatApp) Run() {
	// Show the window first to establish proper sizing for dialogs
	c.myWindow.Show()

	// Now check for loading errors after window is shown
	if err := c.LoadChatHistory(); err != nil {
		dialog.ShowError(err, c.myWindow)
	}

	// Set up model selection after window is shown
	c.SetupModelSelection()

	// Use Run() instead of ShowAndRun() since we already called Show()
	c.myApp.Run()
}

func fetchModels(client *http.Client) ([]Model, error) {
	resp, err := client.Get("http://localhost:11434/api/tags")
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
	client := &http.Client{}
	models, err := fetchModels(client)
	if err != nil {
		dialog.ShowError(err, c.myWindow)
		return
	}

	modelNames := c.extractModelNames(models)

	// Update the existing modelSelect widget instead of creating a new one
	c.modelSelect.Options = modelNames
	c.modelSelect.OnChanged = c.onModelSelect
	c.modelSelect.Refresh()

	savedModel := c.myApp.Preferences().StringWithFallback("selectedModel", "llama3.2:latest")
	c.modelSelect.SetSelected(savedModel)

	// Refresh the parent container to ensure the updated modelSelect is displayed
	c.myWindow.Content().Refresh()
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

func (c *ChatApp) addMessageCard(content string, isUserMessage, saveToHistory bool) *widget.Card {
	var title, sender string
	if isUserMessage {
		title = "You:"
		sender = "user"
	} else {
		title = "LLM:"
		sender = "llm"
	}

	richText := widget.NewRichTextFromMarkdown(content)
	richText.Wrapping = fyne.TextWrapWord
	messageCard := widget.NewCard(title, "", richText)
	c.chatContainer.Add(messageCard)
	c.enableUtilityButtons()

	if saveToHistory {
		c.messages = append(c.messages, ChatMessage{
			Sender:    sender,
			Content:   content,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		if err := c.SaveChatHistory(); err != nil {
			dialog.ShowError(err, c.myWindow)
		}
	}

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

	c.addMessageCard(query, true, true)
	c.inputField.SetText("")

	c.scrollContainer.ScrollToBottom() // Scroll to the bottom to show the new message

	c.showProcessingStatus()
	// scrollContainer := container.NewScroll(c.chatContainer)
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelFunc = cancelFunc

	fullPrompt := c.buildPromptWithHistory(query, 10) // Use the last 10 messages for context
	go c.sendMessageToLLM(ctx, selectedModel, fullPrompt)
}

func (c *ChatApp) sendMessageToLLM(ctx context.Context, selectedModel, query string) {
	var card *widget.Card
	llmResponse := ""
	llmMsgIndex := -1 // Track the index of the LLM message in c.messages

	shouldAutoScroll := func() bool {
		// If the scroll is near the bottom (within 50px), auto-scroll
		offset := c.scrollContainer.Offset.Y
		maxOffset := c.scrollContainer.Content.Size().Height - c.scrollContainer.Size().Height
		return offset >= maxOffset-50
	}

	err := sendQueryToLLM(ctx, selectedModel, query, c.updateStatus, func(chunk string, newStream bool) {
		autoScroll := shouldAutoScroll()
		if newStream {
			llmResponse = chunk
			card = c.addMessageCard(llmResponse, false, true)
			llmMsgIndex = len(c.messages) - 1 // Update the index of the LLM message
		} else {
			llmResponse += chunk
			c.updateRichText(card, llmResponse)

			// Update the correct LLM message in c.messages
			if llmMsgIndex >= 0 && llmMsgIndex < len(c.messages) {
				c.messages[llmMsgIndex].Content = llmResponse
			}
		}

		card.Refresh()
		c.chatContainer.Refresh()
		if autoScroll {
			c.scrollContainer.ScrollToBottom()
		}
	})

	// After streaming is done, ensure the last LLM message is fully saved
	if llmMsgIndex >= 0 && llmMsgIndex < len(c.messages) {
		c.messages[llmMsgIndex].Content = llmResponse
		if err := c.SaveChatHistory(); err != nil {
			dialog.ShowError(err, c.myWindow)
		}
	}
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
		c.addMessageCard("\n**Error:** "+err.Error(), false, false)
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
		c.addMessageCard("\n\n**Request canceled**", false, false)
		c.clearProcessingStatus()
	}

	c.queryInProgress = false // Reset the flag on cancel
	c.updateSendButtonState() // Update the button states
}

func (c *ChatApp) onClearButtonTapped() {
	c.chatContainer.Objects = nil
	c.chatContainer.Refresh()

	c.messages = []ChatMessage{} // Clear the messages slice

	if err := c.SaveChatHistory(); err != nil {
		dialog.ShowError(err, c.myWindow)
	}

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
		defer writer.Close()

		// Save directly from c.messages instead of extracting from UI
		for _, msg := range c.messages {
			var prefix string
			if msg.Sender == "user" {
				prefix = "You:"
			} else {
				prefix = "LLM:"
			}

			content := fmt.Sprintf("%s\n%s\n\n", prefix, msg.Content)
			writer.Write([]byte(content))
		}

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

func (c *ChatApp) buildPromptWithHistory(newUserMessage string, maxMessages int) string {
	var prompt strings.Builder
	prompt.WriteString("You are a helpful assistant.\n\n")

	// Add the last `maxMessages` messages from the history
	start := len(c.messages) - maxMessages
	if start < 0 {
		start = 0
	}

	for i := start; i < len(c.messages); i++ {
		msg := c.messages[i]
		if msg.Sender == "user" {
			prompt.WriteString("user: " + msg.Content + "\n")
		} else {
			prompt.WriteString("llm: " + msg.Content + "\n")
		}
	}

	prompt.WriteString("user: " + newUserMessage + "\nllm:")
	return prompt.String()
}
