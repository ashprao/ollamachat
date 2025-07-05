package ui

import (
	"context"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ashprao/ollamachat/internal/config"
	"github.com/ashprao/ollamachat/internal/models"
	"github.com/ashprao/ollamachat/internal/validation"
	"github.com/ashprao/ollamachat/pkg/logger"
)

// SettingsDialog creates and shows a settings dialog
type SettingsDialog struct {
	logger *logger.Logger
	config *config.Config
	app    AppInterface // Interface to app for saving config
	window fyne.Window
	chatUI *ChatUI // Reference to ChatUI for saving session changes

	// Global settings
	windowWidthEntry    *widget.Entry
	windowHeightEntry   *widget.Entry
	maxMessagesEntry    *widget.Entry
	fontSizeEntry       *widget.Entry
	showTimestampsCheck *widget.Check
	sidebarWidthEntry   *widget.Entry

	// Session-specific settings
	session                 *models.ChatSession
	modelSelect             *widget.Select
	temperatureEntry        *widget.Entry
	sessionMaxMessagesEntry *widget.Entry
}

// NewSettingsDialog creates a new settings dialog
func NewSettingsDialog(window fyne.Window, config *config.Config, session *models.ChatSession, logger *logger.Logger, availableModels []string, chatUI *ChatUI, app AppInterface) *SettingsDialog {
	return &SettingsDialog{
		logger:                  logger.WithComponent("settings-dialog"),
		config:                  config,
		app:                     app,
		window:                  window,
		session:                 session,
		chatUI:                  chatUI,
		windowWidthEntry:        widget.NewEntry(),
		windowHeightEntry:       widget.NewEntry(),
		maxMessagesEntry:        widget.NewEntry(),
		fontSizeEntry:           widget.NewEntry(),
		showTimestampsCheck:     widget.NewCheck("Show timestamps in chat", nil),
		sidebarWidthEntry:       widget.NewEntry(),
		modelSelect:             widget.NewSelect(availableModels, nil),
		temperatureEntry:        widget.NewEntry(),
		sessionMaxMessagesEntry: widget.NewEntry(),
	}
}

// Show displays the settings dialog
func (sd *SettingsDialog) Show() {
	sd.populateCurrentValues()
	sd.setupRealTimeValidation()

	// Create tabs for different settings categories
	globalTab := sd.createGlobalSettingsTab()
	sessionTab := sd.createSessionSettingsTab()

	tabs := container.NewAppTabs(
		container.NewTabItem("Global Settings", globalTab),
		container.NewTabItem("Session Settings", sessionTab),
	)

	// Use ShowCustomConfirm with proper callback
	dialog.ShowCustomConfirm(
		"Settings",
		"Save",   // confirm button text
		"Cancel", // dismiss button text
		tabs,     // content
		func(confirmed bool) {
			if confirmed {
				// User clicked Save
				if err := sd.saveSettings(); err != nil {
					dialog.ShowError(err, sd.window)
					return
				}
				dialog.ShowInformation("Settings Saved", "Settings have been saved successfully", sd.window)
			}
			// User clicked Cancel or confirmed - dialog closes automatically
		},
		sd.window, // parent window
	)
}

// createGlobalSettingsTab creates the global settings tab content
func (sd *SettingsDialog) createGlobalSettingsTab() *fyne.Container {
	windowGroup := widget.NewCard("Window Settings", "",
		container.NewGridWithColumns(2,
			widget.NewLabel("Width:"), sd.windowWidthEntry,
			widget.NewLabel("Height:"), sd.windowHeightEntry,
			widget.NewLabel("Sidebar Width:"), sd.sidebarWidthEntry,
		),
	)

	uiGroup := widget.NewCard("Chat Settings", "",
		container.NewVBox(
			container.NewGridWithColumns(2,
				widget.NewLabel("Font Size:"), sd.fontSizeEntry,
				widget.NewLabel("Default Max Messages:"), sd.maxMessagesEntry,
			),
			sd.showTimestampsCheck,
		),
	)

	return container.NewVBox(windowGroup, uiGroup)
}

// createSessionSettingsTab creates the session-specific settings tab content
func (sd *SettingsDialog) createSessionSettingsTab() *fyne.Container {
	if sd.session == nil {
		return container.NewVBox(widget.NewLabel("No session selected"))
	}

	sessionInfo := widget.NewCard("Session Information", "",
		container.NewVBox(
			widget.NewLabel("Session: "+sd.session.Name),
			widget.NewLabel("ID: "+sd.session.ID),
			widget.NewLabel("Created: "+sd.session.CreatedAt.Format("2006-01-02 15:04:05")),
		),
	)

	// Get current global model for display
	var globalModel string
	if sd.chatUI != nil && sd.chatUI.modelSelect != nil {
		ctx := context.Background()
		prefs, err := sd.chatUI.storage.LoadAppPreferences(ctx)
		if err == nil && prefs.DefaultModel != "" {
			globalModel = prefs.DefaultModel
		} else {
			globalModel = getDefaultModel() // Use centralized default
		}
	}

	// Create info about current model usage
	var modelInfo *widget.Label
	if sd.session.Model != "" {
		modelInfo = widget.NewLabel(fmt.Sprintf("This session uses: %s (session-specific)\nGlobal model: %s", sd.session.Model, globalModel))
	} else {
		modelInfo = widget.NewLabel(fmt.Sprintf("This session uses the global model: %s\nSelect a model below to set a session-specific preference.", globalModel))
	}
	modelInfo.Wrapping = fyne.TextWrapWord

	sessionSettings := widget.NewCard("Session Settings", "",
		container.NewVBox(
			modelInfo,
			container.NewGridWithColumns(2,
				widget.NewLabel("Session-specific Model:"), sd.modelSelect,
				widget.NewLabel("Max Context Messages:"), sd.sessionMaxMessagesEntry,
				widget.NewLabel("Temperature:"), sd.temperatureEntry,
			),
		),
	)

	return container.NewVBox(sessionInfo, sessionSettings)
}

// populateCurrentValues fills the form fields with current values
func (sd *SettingsDialog) populateCurrentValues() {
	// Global settings
	sd.windowWidthEntry.SetText(strconv.Itoa(sd.config.UI.WindowWidth))
	sd.windowHeightEntry.SetText(strconv.Itoa(sd.config.UI.WindowHeight))
	sd.maxMessagesEntry.SetText(strconv.Itoa(sd.config.UI.MaxMessages))
	sd.fontSizeEntry.SetText(strconv.Itoa(sd.config.UI.FontSize))
	sd.showTimestampsCheck.SetChecked(sd.config.UI.ShowTimestamps)
	sd.sidebarWidthEntry.SetText(strconv.Itoa(sd.config.UI.SidebarWidth))

	// Session settings
	if sd.session != nil {
		// For the model selector, show session-specific model if set, otherwise empty (to allow setting one)
		sd.modelSelect.SetSelected(sd.session.Model) // This will be empty string if no session-specific model
		sd.sessionMaxMessagesEntry.SetText(strconv.Itoa(sd.session.MaxMessages))
		sd.temperatureEntry.SetText(fmt.Sprintf("%.2f", sd.session.Temperature))
	}
}

// saveSettings saves the updated settings
// parseAndValidateFields parses and validates all form fields, returning the parsed values
func (sd *SettingsDialog) parseAndValidateFields() (map[string]interface{}, error) {
	values := make(map[string]interface{})

	// Parse and validate window width
	windowWidth, err := validation.ValidatePositiveInt(sd.windowWidthEntry.Text, "window width")
	if err != nil {
		return nil, err
	}
	values["windowWidth"] = windowWidth

	// Parse and validate window height
	windowHeight, err := validation.ValidatePositiveInt(sd.windowHeightEntry.Text, "window height")
	if err != nil {
		return nil, err
	}
	values["windowHeight"] = windowHeight

	// Parse and validate max messages (allow 0 to disable context)
	maxMessages, err := validation.ValidateNonNegativeInt(sd.maxMessagesEntry.Text, "max messages")
	if err != nil {
		return nil, fmt.Errorf("max messages must be a non-negative number (0 to disable context)")
	}
	values["maxMessages"] = maxMessages

	// Parse and validate font size
	fontSize, err := validation.ValidatePositiveInt(sd.fontSizeEntry.Text, "font size")
	if err != nil {
		return nil, err
	}
	values["fontSize"] = fontSize

	// Parse and validate sidebar width
	sidebarWidth, err := validation.ValidatePositiveInt(sd.sidebarWidthEntry.Text, "sidebar width")
	if err != nil {
		return nil, err
	}
	values["sidebarWidth"] = sidebarWidth

	// Parse session-specific settings if session exists
	if sd.session != nil {
		// Parse and validate session max messages (allow 0 to disable context)
		sessionMaxMessages, err := validation.ValidateNonNegativeInt(sd.sessionMaxMessagesEntry.Text, "session max messages")
		if err != nil {
			return nil, fmt.Errorf("session max messages must be a non-negative number (0 to disable context)")
		}
		values["sessionMaxMessages"] = sessionMaxMessages

		// Parse and validate temperature (should be between 0 and 2)
		temperature, err := validation.ValidateFloat(sd.temperatureEntry.Text, "temperature", 0, 2)
		if err != nil {
			return nil, err
		}
		values["temperature"] = temperature
	}

	return values, nil
}

// saveSettings saves the current settings to the configuration
func (sd *SettingsDialog) saveSettings() error {
	// Parse and validate all fields first
	values, err := sd.parseAndValidateFields()
	if err != nil {
		return err
	}

	// Update global settings with validated values
	sd.config.UI.WindowWidth = values["windowWidth"].(int)
	sd.config.UI.WindowHeight = values["windowHeight"].(int)
	sd.config.UI.MaxMessages = values["maxMessages"].(int)
	sd.config.UI.FontSize = values["fontSize"].(int)
	sd.config.UI.SidebarWidth = values["sidebarWidth"].(int)

	sd.config.UI.ShowTimestamps = sd.showTimestampsCheck.Checked

	// Validate the updated configuration
	if err := sd.config.ValidateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save configuration to file
	if err := sd.app.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration to file: %w", err)
	}

	// Apply window size changes immediately
	sd.app.UpdateWindowSize(sd.config.UI.WindowWidth, sd.config.UI.WindowHeight)

	// Apply sidebar width changes immediately
	if sd.chatUI != nil {
		sd.chatUI.UpdateSidebarWidth(sd.config.UI.SidebarWidth)
	}

	// Apply font size changes immediately if possible
	if sd.config.UI.FontSize > 0 {
		sd.app.UpdateFontSize(sd.config.UI.FontSize)
	}

	// Refresh message display if timestamp setting changed
	if sd.chatUI != nil {
		sd.chatUI.RefreshMessageDisplay()
	}

	// Save session settings
	if sd.session != nil {
		maxMessages := values["sessionMaxMessages"].(int)
		temperature := values["temperature"].(float64)

		sd.session.UpdateSessionSettings(
			sd.modelSelect.Selected,
			DefaultProvider, // Use default provider constant
			maxMessages,
			temperature,
		)

		sd.logger.Info("Updated session settings", "session_id", sd.session.ID, "model", sd.modelSelect.Selected, "max_messages", maxMessages, "temperature", temperature)

		// Save the updated session
		if sd.chatUI != nil {
			if err := sd.chatUI.saveCurrentSession(); err != nil {
				return fmt.Errorf("failed to save session: %w", err)
			}

			// Update the main UI model selector if model changed
			sd.chatUI.UpdateModelSelection(sd.modelSelect.Selected)

			// Force refresh of the sessions list to ensure the updated session is reflected
			sd.chatUI.refreshSessionsList()
		}
	}

	sd.logger.Info("Settings saved successfully")
	return nil
}

// validateFields performs real-time validation on all fields
func (sd *SettingsDialog) validateFields() error {
	// Use the shared parsing method for validation
	_, err := sd.parseAndValidateFields()
	return err
}

// setupRealTimeValidation sets up real-time validation for all input fields
func (sd *SettingsDialog) setupRealTimeValidation() {
	// Add validation callback to numeric fields
	validateCallback := func() {
		if err := sd.validateFields(); err != nil {
			sd.logger.Debug("Validation error", "error", err)
			// Could show validation status in UI here
		}
	}

	sd.windowWidthEntry.OnChanged = func(string) { validateCallback() }
	sd.windowHeightEntry.OnChanged = func(string) { validateCallback() }
	sd.maxMessagesEntry.OnChanged = func(string) { validateCallback() }
	sd.fontSizeEntry.OnChanged = func(string) { validateCallback() }
	sd.sidebarWidthEntry.OnChanged = func(string) { validateCallback() }
	sd.sessionMaxMessagesEntry.OnChanged = func(string) { validateCallback() }
	sd.temperatureEntry.OnChanged = func(string) { validateCallback() }
}
