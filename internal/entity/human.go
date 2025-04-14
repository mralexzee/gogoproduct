package entity

// Human extends the base Entity interface with human-specific capabilities
type Human interface {
	Entity

	// Human-specific methods
	Email() string         // Get the email address
	SetEmail(email string) // Set the email address

	// Authentication methods
	IsAuthenticated() bool // Check if the human is authenticated
	SetAuthenticated(bool) // Set authentication status

	// Notification preferences
	NotificationPreferences() map[string]bool           // Get notification preferences
	SetNotificationPreference(key string, enabled bool) // Set a notification preference

	// UI/UX preferences
	ThemePreference() string         // Get theme preference
	SetThemePreference(theme string) // Set theme preference

	// Communication preferences
	PreferredCommunicationChannel() string           // Get preferred communication channel
	SetPreferredCommunicationChannel(channel string) // Set preferred communication channel
}
