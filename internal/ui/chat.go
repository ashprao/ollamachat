package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ashprao/ollamachat/internal/llm"
	"github.com/ashprao/ollamachat/internal/models"
	"github.com/ashprao/ollamachat/internal/storage"
	"github.com/ashprao/ollamachat/pkg/logger"
)

// ChatUI handles the main chat interface
type ChatUI struct {
	// Dependencies
	provider llm.Provider
	storage  storage.Storage
	logger   *logger.Logger

	// UI components
	window          fyne.Window
	chatContainer   *fyne.Container
	scrollContainer *container.Scroll
	inputField      *widget.Entry
	statusLabel     *widget.Label
	modelSelect     *widget.Select
	sendButton      *widget.Button
	clearButton     *widget.Button
	saveButton      *widget.Button
	cancelButton    *widget.Button

	// State
	cancelFunc      context.CancelFunc
	queryInProgress bool
	currentSession  models.ChatSession
}

// NewChatUI creates a new chat UI instance
func NewChatUI(window fyne.Window, provider llm.Provider, storage storage.Storage, logger *logger.Logger) *ChatUI {
	ui := &ChatUI{
		provider:       provider,
		storage:        storage,
		logger:         logger.WithComponent("chat-ui"),
		window:         window,
		chatContainer:  container.NewVBox(),
		inputField:     widget.NewMultiLineEntry(),
		statusLabel:    widget.NewLabel(""),
		modelSelect:    widget.NewSelect([]string{}, nil),
		currentSession: models.NewChatSession("Default Session", ""),
	}

	ui.currentSession.ID = "default" // For backward compatibility
	ui.logger.Info("Chat UI created")
	return ui
}

// Initialize sets up the UI components and loads initial data
func (ui *ChatUI) Initialize() error {
	ui.logger.Info("Initializing chat UI")

	// Load existing session or create new one
	if err := ui.loadCurrentSession(); err != nil {
		ui.logger.Error("Failed to load session", "error", err)
		// Continue with empty session rather than failing
	}

	// Setup UI components
	ui.setupUI()

	// Load available models
	if err := ui.setupModelSelection(); err != nil {
		ui.logger.Error("Failed to setup model selection", "error", err)
		return fmt.Errorf("failed to setup model selection: %w", err)
	}

	ui.logger.Info("Chat UI initialized successfully")
	return nil
}

// setupUI initializes all UI components and layout
func (ui *ChatUI) setupUI() {
	ui.inputField.SetPlaceHolder("Type your query here...")
	ui.inputField.Wrapping = fyne.TextWrapWord
	ui.inputField.OnChanged = ui.onInputFieldChanged

	ui.initButtons()

	// Create model selection container
	modelSelectContainer := container.NewGridWrap(
		fyne.NewSize(ui.calculateModelSelectWidth()+50, ui.modelSelect.MinSize().Height),
		ui.modelSelect,
	)

	// Status area with cancel button
	statusArea := container.NewBorder(nil, nil, nil, ui.cancelButton, ui.statusLabel)

	// Button area
	buttons := container.NewVBox(ui.sendButton, ui.clearButton, ui.saveButton, ui.createQuitButton())

	// Input area
	inputArea := container.NewBorder(nil, nil, nil, buttons, ui.inputField)

	// Scroll container for messages
	ui.scrollContainer = container.NewScroll(ui.chatContainer)
	ui.scrollContainer.SetMinSize(fyne.NewSize(400, 300))

	// Load existing messages into UI
	for _, msg := range ui.currentSession.Messages {
		isUserMessage := msg.Sender == "user"
		ui.addMessageCard(msg.Content, isUserMessage, false)
	}

	// Main content layout
	content := container.NewBorder(
		statusArea,
		container.NewVBox(
			container.NewHBox(modelSelectContainer, widget.NewLabel("Model to be used.")),
			inputArea,
		),
		nil,
		nil,
		ui.scrollContainer,
	)

	ui.window.SetContent(content)
}

// setupModelSelection loads available models and sets up the model selector
func (ui *ChatUI) setupModelSelection() error {
	ctx := context.Background()
	models, err := ui.provider.GetModels(ctx)
	if err != nil {
		ui.logger.Error("Failed to fetch models", "error", err)
		return err
	}

	modelNames := ui.extractModelNames(models)
	ui.modelSelect.Options = modelNames
	ui.modelSelect.OnChanged = ui.onModelSelect
	ui.modelSelect.Refresh()

	// Load saved model preference
	prefs, err := ui.storage.LoadAppPreferences(ctx)
	if err != nil {
		ui.logger.Warn("Failed to load preferences, using default model", "error", err)
		prefs = storage.NewDefaultAppPreferences()
	}

	// Set selected model
	savedModel := prefs.DefaultModel
	if savedModel == "" {
		savedModel = "llama3.2:latest"
	}
	ui.modelSelect.SetSelected(savedModel)

	ui.window.Content().Refresh()
	ui.logger.Info("Model selection setup completed", "model_count", len(models), "selected_model", savedModel)
	return nil
}

// initButtons creates and configures all buttons
func (ui *ChatUI) initButtons() {
	ui.sendButton = widget.NewButtonWithIcon("Send", theme.ConfirmIcon(), ui.onSendButtonTapped)
	ui.clearButton = widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), ui.onClearButtonTapped)
	ui.saveButton = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), ui.onSaveButtonTapped)
	ui.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), ui.onCancelButtonTapped)
	ui.cancelButton.Hide()
	ui.disableUtilityButtons()
}

// loadCurrentSession loads the current chat session from storage
func (ui *ChatUI) loadCurrentSession() error {
	ctx := context.Background()
	session, err := ui.storage.LoadChatSession(ctx, "default")
	if err != nil {
		ui.logger.Info("No existing session found, starting with empty session")
		return nil
	}

	ui.currentSession = session
	ui.logger.Info("Loaded existing session", "message_count", len(session.Messages))
	return nil
}

// saveCurrentSession saves the current chat session to storage
func (ui *ChatUI) saveCurrentSession() error {
	ctx := context.Background()
	if err := ui.storage.SaveChatSession(ctx, ui.currentSession); err != nil {
		ui.logger.Error("Failed to save session", "error", err)
		return err
	}
	return nil
}

// Event handlers

func (ui *ChatUI) onInputFieldChanged(content string) {
	ui.updateSendButtonState()
}

func (ui *ChatUI) onSendButtonTapped() {
	if ui.queryInProgress {
		return
	}

	query := ui.inputField.Text
	selectedModel := ui.modelSelect.Selected
	if query == "" {
		return
	}

	ui.queryInProgress = true
	ui.updateSendButtonState()

	// Add user message to UI and session
	ui.addMessageCard(query, true, true)
	ui.inputField.SetText("")

	ui.scrollContainer.ScrollToBottom()
	ui.showProcessingStatus()

	// Create context for cancellation
	ctx, cancelFunc := context.WithCancel(context.Background())
	ui.cancelFunc = cancelFunc

	// Build prompt with history
	fullPrompt := ui.buildPromptWithHistory(query, 10)
	go ui.sendMessageToLLM(ctx, selectedModel, fullPrompt)
}

func (ui *ChatUI) onClearButtonTapped() {
	ui.chatContainer.Objects = nil
	ui.chatContainer.Refresh()

	ui.currentSession.Messages = []models.ChatMessage{}
	if err := ui.saveCurrentSession(); err != nil {
		dialog.ShowError(err, ui.window)
	}

	ui.disableUtilityButtons()
	ui.window.Content().Refresh()
	ui.logger.Info("Chat cleared")
}

func (ui *ChatUI) onSaveButtonTapped() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}

		filename := writer.URI().Name()
		if !strings.HasSuffix(filename, ".txt") {
			filename += ".txt"
		}
		defer writer.Close()

		// Save chat history to file
		for _, msg := range ui.currentSession.Messages {
			var prefix string
			if msg.Sender == "user" {
				prefix = "You:"
			} else {
				prefix = "LLM:"
			}

			content := fmt.Sprintf("%s\n%s\n\n", prefix, msg.Content)
			writer.Write([]byte(content))
		}

		ui.saveButton.Disable()
		ui.logger.Info("Chat saved to file", "filename", filename)
	}, ui.window)
}

func (ui *ChatUI) onCancelButtonTapped() {
	if ui.cancelFunc != nil {
		ui.cancelFunc()
		ui.addMessageCard("\n\n**Request canceled**", false, false)
		ui.clearProcessingStatus()
	}

	ui.queryInProgress = false
	ui.updateSendButtonState()
}

func (ui *ChatUI) onModelSelect(selected string) {
	ctx := context.Background()
	prefs, err := ui.storage.LoadAppPreferences(ctx)
	if err != nil {
		prefs = storage.NewDefaultAppPreferences()
	}

	prefs.DefaultModel = selected
	if err := ui.storage.SaveAppPreferences(ctx, prefs); err != nil {
		ui.logger.Error("Failed to save model preference", "error", err)
	}

	ui.logger.Info("Model selected", "model", selected)
}

// Helper methods

func (ui *ChatUI) extractModelNames(models []models.Model) []string {
	modelNames := make([]string, len(models))
	for i, model := range models {
		modelNames[i] = model.Name
	}
	return modelNames
}

func (ui *ChatUI) calculateModelSelectWidth() float32 {
	if ui.modelSelect == nil || len(ui.modelSelect.Options) == 0 {
		return 100
	}

	longestModel := ""
	for _, model := range ui.modelSelect.Options {
		if len(model) > len(longestModel) {
			longestModel = model
		}
	}
	return canvas.NewText(longestModel, nil).MinSize().Width
}

func (ui *ChatUI) addMessageCard(content string, isUserMessage, saveToHistory bool) *widget.Card {
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
	ui.chatContainer.Add(messageCard)
	ui.enableUtilityButtons()

	if saveToHistory {
		message := models.NewChatMessage(sender, content)
		ui.currentSession.AddMessage(message)
		if err := ui.saveCurrentSession(); err != nil {
			dialog.ShowError(err, ui.window)
		}
	}

	return messageCard
}

func (ui *ChatUI) enableUtilityButtons() {
	ui.clearButton.Enable()
	ui.saveButton.Enable()
}

func (ui *ChatUI) disableUtilityButtons() {
	ui.sendButton.Disable()
	ui.clearButton.Disable()
	ui.saveButton.Disable()
}

func (ui *ChatUI) updateSendButtonState() {
	if ui.queryInProgress || ui.inputField.Text == "" {
		ui.sendButton.Disable()
	} else {
		ui.sendButton.Enable()
	}
}

func (ui *ChatUI) showProcessingStatus() {
	ui.statusLabel.SetText("Processing...")
	ui.cancelButton.Show()
}

func (ui *ChatUI) clearProcessingStatus() {
	ui.statusLabel.SetText("")
	ui.cancelButton.Hide()
	ui.window.Content().Refresh()
}

func (ui *ChatUI) updateStatus(status string) {
	ui.statusLabel.SetText(status)
	ui.window.Content().Refresh()
}

func (ui *ChatUI) updateRichText(card *widget.Card, content string) {
	if richText, ok := card.Content.(*widget.RichText); ok {
		richText.ParseMarkdown(content)
	}
}

func (ui *ChatUI) handleLLMResponseError(err error) {
	if err != nil && err.Error() != "context canceled" {
		ui.addMessageCard("\n**Error:** "+err.Error(), false, false)
	}
	ui.clearProcessingStatus()
}

func (ui *ChatUI) createQuitButton() *widget.Button {
	return widget.NewButtonWithIcon("Quit", theme.LogoutIcon(), func() {
		ui.window.Close()
	})
}

func (ui *ChatUI) buildPromptWithHistory(newUserMessage string, maxMessages int) string {
	var prompt strings.Builder
	prompt.WriteString("You are a helpful assistant.\n\n")

	// Add the last `maxMessages` messages from the history
	messages := ui.currentSession.Messages
	start := len(messages) - maxMessages
	if start < 0 {
		start = 0
	}

	for i := start; i < len(messages); i++ {
		msg := messages[i]
		if msg.Sender == "user" {
			prompt.WriteString("user: " + msg.Content + "\n")
		} else {
			prompt.WriteString("llm: " + msg.Content + "\n")
		}
	}

	prompt.WriteString("user: " + newUserMessage + "\nllm:")
	return prompt.String()
}

// sendMessageToLLM handles the streaming LLM response
func (ui *ChatUI) sendMessageToLLM(ctx context.Context, selectedModel, query string) {
	var card *widget.Card
	llmResponse := ""
	var llmMessage *models.ChatMessage

	shouldAutoScroll := func() bool {
		offset := ui.scrollContainer.Offset.Y
		maxOffset := ui.scrollContainer.Content.Size().Height - ui.scrollContainer.Size().Height
		return offset >= maxOffset-50
	}

	err := ui.provider.SendQuery(ctx, selectedModel, query, func(chunk string, newStream bool) {
		autoScroll := shouldAutoScroll()
		if newStream {
			llmResponse = chunk
			card = ui.addMessageCard(llmResponse, false, false)
			// Create the LLM message and add it to session
			llmMessage = &models.ChatMessage{
				Sender:    "llm",
				Content:   llmResponse,
				Timestamp: time.Now(),
			}
			ui.currentSession.AddMessage(*llmMessage)
		} else {
			llmResponse += chunk
			ui.updateRichText(card, llmResponse)

			// Update the last message in the session
			if len(ui.currentSession.Messages) > 0 {
				lastIdx := len(ui.currentSession.Messages) - 1
				if ui.currentSession.Messages[lastIdx].Sender == "llm" {
					ui.currentSession.Messages[lastIdx].Content = llmResponse
				}
			}
		}

		card.Refresh()
		ui.chatContainer.Refresh()
		if autoScroll {
			ui.scrollContainer.ScrollToBottom()
		}
	})

	// Save the final session state
	if err := ui.saveCurrentSession(); err != nil {
		dialog.ShowError(err, ui.window)
	}

	ui.queryInProgress = false
	ui.updateSendButtonState()
	ui.handleLLMResponseError(err)
}
