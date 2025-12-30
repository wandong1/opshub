package plugin

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Plugin 插件接口
// All plugins must implement this interface
type Plugin interface {
	// Name Plugin unique identifier
	Name() string

	// Description Plugin description
	Description() string

	// Version Plugin version
	Version() string

	// Author Plugin author
	Author() string

	// Enable Enable plugin
	// Initialize plugin resources, database tables, etc.
	Enable(db *gorm.DB) error

	// Disable Disable plugin
	// Clean up plugin resources (note: won't delete database tables by default)
	Disable(db *gorm.DB) error

	// RegisterRoutes Register routes
	// Plugin can register its API routes here
	RegisterRoutes(router *gin.RouterGroup, db *gorm.DB)

	// GetMenus Get plugin menu configuration
	// Return menu items to be added to the system
	GetMenus() []MenuConfig
}

// MenuConfig Menu configuration
type MenuConfig struct {
	// Menu name
	Name string `json:"name"`

	// Menu path (frontend route)
	Path string `json:"path"`

	// Icon name
	Icon string `json:"icon"`

	// Sort order (smaller number comes first)
	Sort int `json:"sort"`

	// Hidden or not
	Hidden bool `json:"hidden"`

	// Parent menu path (if this is a submenu)
	ParentPath string `json:"parentPath"`

	// Permission identifier (optional, for access control)
	Permission string `json:"permission"`
}

// Manager Plugin manager
type Manager struct {
	plugins map[string]Plugin
	db      *gorm.DB
}

// NewManager Create plugin manager
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		db:      db,
	}
}

// Register 注册插件
func (m *Manager) Register(plugin Plugin) error {
	name := plugin.Name()

	// Check if plugin already registered
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Register plugin
	m.plugins[name] = plugin
	return nil
}

// Enable 启用插件
func (m *Manager) Enable(name string) error {
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Execute plugin Enable method
	if err := plugin.Enable(m.db); err != nil {
		return err
	}

	return nil
}

// Disable 禁用插件
func (m *Manager) Disable(name string) error {
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Execute plugin Disable method
	if err := plugin.Disable(m.db); err != nil {
		return err
	}

	return nil
}

// GetPlugin Get plugin
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	plugin, exists := m.plugins[name]
	return plugin, exists
}

// GetAllPlugins Get all plugins
func (m *Manager) GetAllPlugins() []Plugin {
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// RegisterAllRoutes Register all plugin routes
func (m *Manager) RegisterAllRoutes(router *gin.RouterGroup) {
	for _, plugin := range m.plugins {
		// Create route group for each plugin
		pluginGroup := router.Group("/plugins/" + plugin.Name())
		plugin.RegisterRoutes(pluginGroup, m.db)
	}
}

// GetAllMenus Get all plugin menu configurations
func (m *Manager) GetAllMenus() []MenuConfig {
	allMenus := make([]MenuConfig, 0)
	for _, plugin := range m.plugins {
		menus := plugin.GetMenus()
		allMenus = append(allMenus, menus...)
	}
	return allMenus
}
