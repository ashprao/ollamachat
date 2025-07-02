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

// AppInterface defines the interface for app functions that the UI needs
type AppInterface interface {
	SwitchProvider(providerType string) error
	GetAvailableProviders() []string
	GetCurrentProviderType() string
}

// NewChatUI creates a new chat UI instance
func NewChatUI(window fyne.Window, provider llm.Provider, storage storage.Storage, logger *logger.Logger, availableProviders []string, currentProviderType string) *ChatUI {
	ui := &ChatUI{
		provider:            provider,
		storage:             storage,
		logger:              logger.WithComponent("chat-ui"),
		window:              window,
		chatContainer:       container.NewVBox(),
		inputField:          widget.NewMultiLineEntry(),
		statusLabel:         widget.NewLabel(""),
		modelSelect:         widget.NewSelect([]string{}, nil),
		availableProviders:  availableProviders,
		currentProviderType: currentProviderType,
		currentSession:      models.NewChatSession("Default Session", ""),
	}

	ui.currentSession.ID = "default" // For backward compatibility
	ui.logger.Info("Chat UI created")
	return ui
}

// ChatUI handles the main chat interface
type ChatUI struct {
	// Dependencies
	provider            llm.Provider
	storage             storage.Storage
	logger              *logger.Logger
	availableProviders  []string
	currentProviderType string

	// UI components
	window          fyne.Window
	chatContainer   *fyne.Container
	scrollContainer *container.Scroll
	inputField      *widget.Entry
	statusLabel     *widget.Label
	modelSelect     *widget.Select
	providerSelect  *widget.Select
	providerLabel   *widget.Label
	sendButton      *widget.Button
	clearButton     *widget.Button
	saveButton      *widget.Button
	cancelButton    *widget.Button

	// Session management UI
	sessionList      *widget.List
	newSessionButton *widget.Button
	sessionSidebar   *fyne.Container
	sessions         []models.ChatSession

	// State
	cancelFunc      context.CancelFunc
	queryInProgress bool
	currentSession  models.ChatSession
}

// Initialize sets up the UI components and loads initial data
func (ui *ChatUI) Initialize() error {
	ui.logger.Info("Initializing chat UI")

	// Load all sessions
	if err := ui.loadAllSessions(); err != nil {
		ui.logger.Error("Failed to load sessions", "error", err)
		// Continue with empty session rather than failing
	}

	// Load or create current session
	if err := ui.loadCurrentSession(); err != nil {
		ui.logger.Error("Failed to load current session", "error", err)
		// Continue with empty session rather than failing
	}

	// Setup UI components
	ui.setupUI()

	// Load messages from current session into UI
	ui.loadCurrentSessionMessages()

	// Load available models
	if err := ui.setupModelSelection(); err != nil {
		ui.logger.Error("Failed to setup model selection", "error", err)
		return fmt.Errorf("failed to setup model selection: %w", err)
	}

	// Select current session in the sidebar
	ui.selectCurrentSessionInList()

	ui.logger.Info("Chat UI initialized successfully")
	return nil
}

// setupUI initializes all UI components and layout
func (ui *ChatUI) setupUI() {
	ui.inputField.SetPlaceHolder("Type your query here...")
	ui.inputField.Wrapping = fyne.TextWrapWord
	ui.inputField.OnChanged = ui.onInputFieldChanged

	ui.initButtons()
	ui.initProviderUI()
	ui.setupSessionSidebar()
	ui.initProviderUI()

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

	// Note: Messages will be loaded after currentSession is set

	// Main content layout with session sidebar
	chatContent := container.NewBorder(
		statusArea,
		container.NewVBox(
			container.NewHBox(ui.providerLabel, modelSelectContainer, widget.NewLabel("Model to be used.")),
			inputArea,
		),
		nil,
		nil,
		ui.scrollContainer,
	)

	// Main layout with resizable session sidebar
	mainContent := container.NewHSplit(ui.sessionSidebar, chatContent)
	mainContent.SetOffset(0.25) // Sidebar takes 25% of width initially

	ui.window.SetContent(mainContent)
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
	ui.clearButton = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), ui.onClearButtonTapped)
	ui.saveButton = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), ui.onSaveButtonTapped)
	ui.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), ui.onCancelButtonTapped)
	ui.cancelButton.Hide()
	ui.disableUtilityButtons()
}

// initProviderUI initializes provider-related UI components
func (ui *ChatUI) initProviderUI() {
	// Provider label to show current provider
	ui.providerLabel = widget.NewLabel(fmt.Sprintf("Provider: %s", ui.currentProviderType))
	ui.providerLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Provider selector (for future use when multiple providers are available)
	ui.providerSelect = widget.NewSelect(ui.availableProviders, ui.onProviderSelected)
	ui.providerSelect.SetSelected(ui.currentProviderType)

	// For now, hide the selector since we only have Ollama
	if len(ui.availableProviders) <= 1 {
		ui.providerSelect.Hide()
	}
}

// loadCurrentSession loads the most recent chat session from storage
func (ui *ChatUI) loadCurrentSession() error {
	// If we have sessions loaded, use the most recent one (first in sorted list)
	if len(ui.sessions) > 0 {
		ui.currentSession = ui.sessions[0]
		ui.logger.Info("Loaded most recent session", "session_id", ui.currentSession.ID, "message_count", len(ui.currentSession.Messages))
		return nil
	}

	// No existing sessions, create a new one
	ui.currentSession = models.NewChatSession(fmt.Sprintf("Session %s", time.Now().Format("15:04")), "")
	ui.autoSaveCurrentSession()
	ui.logger.Info("Created new session", "session_id", ui.currentSession.ID)
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
	// Show confirmation dialog for session deletion
	dialog.ShowConfirm("Delete Session",
		fmt.Sprintf("Are you sure you want to delete the session '%s'? This action cannot be undone.", ui.currentSession.Name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			ui.deleteCurrentSession()
		}, ui.window)
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

// onProviderSelected handles provider selection changes
func (ui *ChatUI) onProviderSelected(selected string) {
	if selected == ui.currentProviderType {
		return // No change
	}

	ui.logger.Info("Provider selection changed", "from", ui.currentProviderType, "to", selected)

	// For now, just show a message that this feature will be available later
	dialog.ShowInformation("Provider Switching",
		fmt.Sprintf("Switching to %s provider will be available in a future version.", selected),
		ui.window)
}

// UpdateProvider updates the UI to use a new provider
func (ui *ChatUI) UpdateProvider(newProvider llm.Provider) {
	ui.provider = newProvider

	// Update current provider type based on provider name
	providerName := newProvider.GetName()
	if strings.Contains(strings.ToLower(providerName), "ollama") {
		ui.currentProviderType = "ollama"
	} else if strings.Contains(strings.ToLower(providerName), "openai") {
		ui.currentProviderType = "openai"
	} else if strings.Contains(strings.ToLower(providerName), "eino") {
		ui.currentProviderType = "eino"
	}

	ui.logger.Info("UI updated with new provider", "provider", newProvider.GetName(), "type", ui.currentProviderType)

	// Reload models for the new provider
	if err := ui.setupModelSelection(); err != nil {
		ui.logger.Error("Failed to reload models for new provider", "error", err)
	}

	// Update any provider-specific UI elements
	ui.refreshProviderInfo()
}

// refreshProviderInfo updates UI elements that display provider information
func (ui *ChatUI) refreshProviderInfo() {
	if ui.providerLabel != nil {
		ui.providerLabel.SetText(fmt.Sprintf("Provider: %s", ui.currentProviderType))
	}
	if ui.providerSelect != nil {
		ui.providerSelect.SetSelected(ui.currentProviderType)
	}
	ui.logger.Info("Provider info refreshed", "provider", ui.provider.GetName())
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

		// Update session model if user message and model is selected
		if isUserMessage && ui.modelSelect.Selected != "" {
			ui.currentSession.Model = ui.modelSelect.Selected
		}

		// Auto-save session
		ui.autoSaveCurrentSession()

		// Refresh sessions list to show updated timestamp
		go ui.refreshSessionsList()
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

// Session Management Methods

// loadAllSessions loads all available chat sessions from storage
func (ui *ChatUI) loadAllSessions() error {
	ctx := context.Background()
	sessions, err := ui.storage.ListChatSessions(ctx)
	if err != nil {
		ui.logger.Error("Failed to list sessions", "error", err)
		// Initialize with empty sessions list
		ui.sessions = []models.ChatSession{}
		return nil
	}

	ui.sessions = sessions
	ui.logger.Info("Loaded sessions", "count", len(sessions))

	// Debug logging for UI session order
	ui.logger.Info("UI Session order:")
	for i, session := range ui.sessions {
		ui.logger.Info("UI Session", "index", i, "id", session.ID, "name", session.Name, "updated_at", session.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// setupSessionSidebar creates and configures the session management sidebar
func (ui *ChatUI) setupSessionSidebar() {
	// New Session button
	ui.newSessionButton = widget.NewButton("+ New Session", ui.onNewSessionTapped)
	ui.newSessionButton.Importance = widget.HighImportance

	// Session list
	ui.sessionList = widget.NewList(
		func() int {
			return len(ui.sessions)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("Session Name"),
				widget.NewLabel("Last Updated"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(ui.sessions) {
				return
			}

			session := ui.sessions[id]
			container := obj.(*fyne.Container)
			nameLabel := container.Objects[0].(*widget.Label)
			timeLabel := container.Objects[1].(*widget.Label)

			nameLabel.SetText(session.Name)
			timeLabel.SetText(session.UpdatedAt.Format("Jan 2, 15:04"))

			// Highlight current session
			if session.ID == ui.currentSession.ID {
				nameLabel.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				nameLabel.TextStyle = fyne.TextStyle{}
			}

			nameLabel.Refresh()
		},
	)

	ui.sessionList.OnSelected = ui.onSessionSelected

	// Session sidebar container with elegant border
	sidebarTitle := widget.NewLabel("Chat Sessions")
	sidebarTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Create sidebar content
	sidebarContent := container.NewBorder(
		container.NewVBox(sidebarTitle, ui.newSessionButton),
		nil,
		nil,
		nil,
		ui.sessionList,
	)

	// Add padding and create a container with visual separation
	sidebarWithPadding := container.NewPadded(sidebarContent)

	// Create a vertical separator line for elegant border
	separator := canvas.NewLine(theme.SeparatorColor())
	separator.StrokeWidth = 1

	ui.sessionSidebar = container.NewBorder(
		nil, nil, nil, separator, sidebarWithPadding,
	)
}

// onNewSessionTapped handles creating a new chat session
func (ui *ChatUI) onNewSessionTapped() {
	// Save current session before switching
	if err := ui.saveCurrentSession(); err != nil {
		ui.logger.Error("Failed to save current session", "error", err)
		dialog.ShowError(err, ui.window)
		return
	}

	// Create new session
	newSession := models.NewChatSession(fmt.Sprintf("Session %s", time.Now().Format("15:04")), ui.modelSelect.Selected)

	// Save the new session immediately
	ui.currentSession = newSession
	ui.autoSaveCurrentSession()

	// Add to beginning of sessions list (most recent first)
	ui.sessions = append([]models.ChatSession{newSession}, ui.sessions...)

	// Clear UI and update session selection
	ui.chatContainer.Objects = nil
	ui.chatContainer.Refresh()

	// Update session selection using helper method
	ui.updateSessionSelection(newSession)

	ui.logger.Info("Created new session", "session_id", newSession.ID, "session_name", newSession.Name)
}

// onSessionSelected handles switching to a selected session
func (ui *ChatUI) onSessionSelected(id widget.ListItemID) {
	if id >= len(ui.sessions) {
		return
	}

	selectedSession := ui.sessions[id]

	// Don't switch if it's the same session
	if selectedSession.ID == ui.currentSession.ID {
		return
	}

	// Save current session before switching
	if err := ui.saveCurrentSession(); err != nil {
		ui.logger.Error("Failed to save current session", "error", err)
		dialog.ShowError(err, ui.window)
		return
	}

	// Switch to selected session using the helper method
	ui.updateSessionSelection(selectedSession)

	// Clear and reload UI
	ui.chatContainer.Objects = nil
	ui.chatContainer.Refresh()

	// Load session messages into UI
	for _, msg := range ui.currentSession.Messages {
		isUserMessage := msg.Sender == "user"
		ui.addMessageCard(msg.Content, isUserMessage, false)
	}

	// Update model selection if session has a saved model
	if ui.currentSession.Model != "" {
		ui.modelSelect.SetSelected(ui.currentSession.Model)
	}

	ui.scrollContainer.ScrollToBottom()

	ui.logger.Info("Switched to session", "session_id", selectedSession.ID, "session_name", selectedSession.Name)
}

// refreshSessionsList updates the sessions list and refreshes the UI
func (ui *ChatUI) refreshSessionsList() {
	if err := ui.loadAllSessions(); err != nil {
		ui.logger.Error("Failed to refresh sessions list", "error", err)
		return
	}
	ui.sessionList.Refresh()

	// Select current session in the list if it exists
	ui.selectCurrentSessionInList()
}

// selectCurrentSessionInList finds and selects the current session in the session list
func (ui *ChatUI) selectCurrentSessionInList() {
	for i, session := range ui.sessions {
		if session.ID == ui.currentSession.ID {
			ui.sessionList.Select(i)
			break
		}
	}
}

// autoSaveCurrentSession saves the current session automatically
func (ui *ChatUI) autoSaveCurrentSession() {
	if err := ui.saveCurrentSession(); err != nil {
		ui.logger.Error("Auto-save failed", "error", err)
		// Don't show dialog for auto-save failures to avoid interrupting user
	}
}

// deleteCurrentSession deletes the current chat session and updates the UI
func (ui *ChatUI) deleteCurrentSession() {
	sessionID := ui.currentSession.ID
	ui.logger.Info("Deleting current session", "session_id", sessionID)

	// Delete the session from storage
	if err := ui.storage.DeleteChatSession(context.Background(), sessionID); err != nil {
		ui.logger.Error("Failed to delete session", "session_id", sessionID, "error", err)
		dialog.ShowError(fmt.Errorf("failed to delete session: %v", err), ui.window)
		return
	}

	// Get updated sessions list
	sessions, err := ui.storage.ListChatSessions(context.Background())
	if err != nil {
		ui.logger.Error("Failed to list sessions after deletion", "error", err)
		dialog.ShowError(fmt.Errorf("failed to refresh sessions: %v", err), ui.window)
		return
	}

	// If no sessions remain, create a new session
	if len(sessions) == 0 {
		ui.logger.Info("No sessions remaining, creating new session")
		newSession := models.NewChatSession(fmt.Sprintf("Session %s", time.Now().Format("15:04")), ui.currentSession.Model)
		ui.currentSession = newSession
		ui.autoSaveCurrentSession()
	} else {
		// Switch to the first available session (most recent due to sorting)
		ui.currentSession = sessions[0]
		ui.logger.Info("Switched to next available session", "session_id", ui.currentSession.ID)
	}

	// Clear and refresh UI
	ui.chatContainer.Objects = nil
	ui.chatContainer.Refresh()

	// Load session messages into UI
	for _, msg := range ui.currentSession.Messages {
		isUserMessage := msg.Sender == "user"
		ui.addMessageCard(msg.Content, isUserMessage, false)
	}

	// Update model selection if session has a saved model
	if ui.currentSession.Model != "" {
		ui.modelSelect.SetSelected(ui.currentSession.Model)
	}

	ui.scrollContainer.ScrollToBottom()

	// Refresh sessions list and automatically select the current session
	go func() {
		ui.refreshSessionsList()
		// Find and select the current session in the list
		for i, session := range ui.sessions {
			if session.ID == ui.currentSession.ID {
				ui.sessionList.Select(i)
				break
			}
		}
	}()

	ui.disableUtilityButtons()
	ui.window.Content().Refresh()
	ui.logger.Info("Session deleted successfully", "deleted_session_id", sessionID, "current_session_id", ui.currentSession.ID)
}

// loadCurrentSessionMessages loads messages from the current session into the UI
func (ui *ChatUI) loadCurrentSessionMessages() {
	// Clear existing messages
	ui.chatContainer.Objects = nil

	// Load messages from current session
	for _, msg := range ui.currentSession.Messages {
		isUserMessage := msg.Sender == "user"
		ui.addMessageCard(msg.Content, isUserMessage, false)
	}

	// Update model selection if session has a saved model
	if ui.currentSession.Model != "" && ui.modelSelect != nil {
		ui.modelSelect.SetSelected(ui.currentSession.Model)
	}

	ui.chatContainer.Refresh()
	if ui.scrollContainer != nil {
		ui.scrollContainer.ScrollToBottom()
	}

	ui.logger.Info("Loaded session messages into UI", "session_id", ui.currentSession.ID, "message_count", len(ui.currentSession.Messages))
}

// updateSessionSelection updates the current session and refreshes the UI properly
func (ui *ChatUI) updateSessionSelection(newSession models.ChatSession) {
	ui.currentSession = newSession

	// Refresh the session list to update bold styling
	ui.sessionList.Refresh()

	// Select the session in the list
	for i, session := range ui.sessions {
		if session.ID == newSession.ID {
			ui.sessionList.Select(i)
			break
		}
	}

	ui.logger.Info("Updated session selection", "session_id", newSession.ID, "session_name", newSession.Name)
}
