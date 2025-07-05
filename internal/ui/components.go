package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ashprao/ollamachat/internal/constants"
)

// ComponentFactory provides reusable UI components
type ComponentFactory struct{}

// NewComponentFactory creates a new component factory
func NewComponentFactory() *ComponentFactory {
	return &ComponentFactory{}
}

// CreateIconButton creates a button with an icon and callback
func (cf *ComponentFactory) CreateIconButton(label string, icon fyne.Resource, callback func()) *widget.Button {
	return widget.NewButtonWithIcon(label, icon, callback)
}

// CreateStatusBar creates a status bar with label and optional cancel button
func (cf *ComponentFactory) CreateStatusBar(statusLabel *widget.Label, cancelButton *widget.Button) *fyne.Container {
	if cancelButton != nil {
		return container.NewBorder(nil, nil, nil, cancelButton, statusLabel)
	}
	return container.NewBorder(nil, nil, nil, nil, statusLabel)
}

// CreateButtonGroup creates a vertical group of buttons
func (cf *ComponentFactory) CreateButtonGroup(buttons ...*widget.Button) *fyne.Container {
	buttonObjs := make([]fyne.CanvasObject, len(buttons))
	for i, btn := range buttons {
		buttonObjs[i] = btn
	}
	return container.NewVBox(buttonObjs...)
}

// CreateInputArea creates an input area with text field and buttons
func (cf *ComponentFactory) CreateInputArea(inputField *widget.Entry, buttons *fyne.Container) *fyne.Container {
	return container.NewBorder(nil, nil, nil, buttons, inputField)
}

// CreateModelSelector creates a model selection container
func (cf *ComponentFactory) CreateModelSelector(modelSelect *widget.Select, width float32) *fyne.Container {
	return container.NewGridWrap(
		fyne.NewSize(width, modelSelect.MinSize().Height),
		modelSelect,
	)
}

// CreateScrollableContent creates a scrollable container for content
func (cf *ComponentFactory) CreateScrollableContent(content *fyne.Container, minWidth, minHeight float32) *container.Scroll {
	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(minWidth, minHeight))
	return scroll
}

// CreateMessageCard creates a message card for chat display
func (cf *ComponentFactory) CreateMessageCard(title, content string, isMarkdown bool) *widget.Card {
	var contentWidget fyne.CanvasObject

	if isMarkdown {
		richText := widget.NewRichTextFromMarkdown(content)
		richText.Wrapping = fyne.TextWrapWord
		contentWidget = richText
	} else {
		label := widget.NewLabel(content)
		label.Wrapping = fyne.TextWrapWord
		contentWidget = label
	}

	return widget.NewCard(title, "", contentWidget)
}

// CreateSeparator creates a separator line
func (cf *ComponentFactory) CreateSeparator() *widget.Separator {
	return widget.NewSeparator()
}

// CreateLabelWithStyle creates a label with specified text style
func (cf *ComponentFactory) CreateLabelWithStyle(text string, bold, italic bool) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle.Bold = bold
	label.TextStyle.Italic = italic
	return label
}

// CreateMainLayout creates the main application layout
func (cf *ComponentFactory) CreateMainLayout(
	statusArea, contentArea, inputArea *fyne.Container,
	scrollContainer *container.Scroll,
) *fyne.Container {
	return container.NewBorder(
		statusArea,
		inputArea,
		nil,
		nil,
		scrollContainer,
	)
}

// StandardButtons provides standard button configurations
type StandardButtons struct {
	Send   ButtonConfig
	Clear  ButtonConfig
	Save   ButtonConfig
	Cancel ButtonConfig
	Quit   ButtonConfig
}

// ButtonConfig holds button configuration
type ButtonConfig struct {
	Label string
	Icon  fyne.Resource
}

// GetStandardButtons returns standard button configurations
func (cf *ComponentFactory) GetStandardButtons() StandardButtons {
	return StandardButtons{
		Send: ButtonConfig{
			Label: "Send",
			Icon:  theme.ConfirmIcon(),
		},
		Clear: ButtonConfig{
			Label: "Clear",
			Icon:  theme.DeleteIcon(),
		},
		Save: ButtonConfig{
			Label: "Save",
			Icon:  theme.DocumentSaveIcon(),
		},
		Cancel: ButtonConfig{
			Label: "Cancel",
			Icon:  theme.CancelIcon(),
		},
		Quit: ButtonConfig{
			Label: "Quit",
			Icon:  theme.LogoutIcon(),
		},
	}
}

// UIMetrics provides standard UI measurements
type UIMetrics struct {
	DefaultWindowWidth     float32
	DefaultWindowHeight    float32
	DefaultScrollMinWidth  float32
	DefaultScrollMinHeight float32
	DefaultButtonSpacing   float32
}

// GetStandardMetrics returns standard UI measurements
func (cf *ComponentFactory) GetStandardMetrics() UIMetrics {
	return UIMetrics{
		DefaultWindowWidth:     float32(constants.DefaultWindowWidth),
		DefaultWindowHeight:    float32(constants.DefaultWindowHeight),
		DefaultScrollMinWidth:  400,
		DefaultScrollMinHeight: 300,
		DefaultButtonSpacing:   5,
	}
}

// CreateProgressIndicator creates a progress indicator
func (cf *ComponentFactory) CreateProgressIndicator(text string) *fyne.Container {
	progress := widget.NewProgressBarInfinite()
	label := widget.NewLabel(text)
	return container.NewVBox(label, progress)
}

// CreateInfoPanel creates an information panel with title and content
func (cf *ComponentFactory) CreateInfoPanel(title, content string) *fyne.Container {
	titleLabel := cf.CreateLabelWithStyle(title, true, false)
	contentLabel := widget.NewLabel(content)
	contentLabel.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		titleLabel,
		cf.CreateSeparator(),
		contentLabel,
	)
}
