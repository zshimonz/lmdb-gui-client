package main

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/lmdb-go/lmdbscan"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/zshimonz/lmdb-gui-client/config"
	mytheme "github.com/zshimonz/lmdb-gui-client/theme"
)

var env *lmdb.Env
var dbi lmdb.DBI
var keyValues []KeyValue
var selectedKey string
var windowWidth float32
var windowHeight float32
var connectionList *widget.List
var keyValueTable *widget.Table

var valueView *widget.Entry
var selectedConnectionIndex = -1
var valueLabelString = binding.NewString()

var connectionsPanel *fyne.Container
var valuePanel *fyne.Container
var leftMainSplit *container.Split

var valuePanelOpen = false
var connectionPanelOpen = true

var logMessage = binding.NewString()
var valueSplitOffset = 0.6

var darkMode = true
var logText *canvas.Text

var tabTitle = binding.NewString()
var tabView *fyne.Container

var newConnectionTabItem *fyne.Container
var editConnectionTabItem *fyne.Container
var newKeyValesTabItem *fyne.Container
var keyValuesTabItem *container.Split

var editConnectionNameEntry *widget.Entry
var editConnectionPathEntry *widget.Entry
var editConnectionMapSizeEntry *widget.Entry
var editConnectionIndex int
var toggleConnectionsButton *widget.Button

var keyPrefix = binding.NewString()
var hideKeyPrefix = binding.NewBool()

type KeyValue struct {
	Key   string
	Value string
}

func main() {
	a := app.New()

	lightTheme := &mytheme.MyLightTheme{}
	darkTheme := &mytheme.MyDarkTheme{}

	a.Settings().SetTheme(darkTheme)

	w := a.NewWindow("LMDB GUI Client")

	// set window icon
	iconResource, err := fyne.LoadResourceFromPath("icon.png")
	if err == nil {
		w.SetIcon(iconResource)
	}

	err = config.LoadConfig()
	if err != nil {
		showErrorLog("Error loading config: " + err.Error())
	}

	// 左侧布局：Connection 列表
	connectionList = widget.NewList(
		func() int { return len(config.Config.Connections) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignLeading
			toolbar := widget.NewToolbar(
				widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {}),
				widget.NewToolbarAction(theme.DeleteIcon(), func() {}),
				widget.NewToolbarAction(theme.ContentClearIcon(), func() { connectionList.UnselectAll() }),
			)

			return container.NewBorder(nil, nil, label, toolbar)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*fyne.Container).Objects[0].(*widget.Label)
			label.SetText(config.Config.Connections[i].Name)

			toolbar := o.(*fyne.Container).Objects[1].(*widget.Toolbar)
			editButton := toolbar.Items[0].(*widget.ToolbarAction)
			editButton.OnActivated = func() {
				if selectedConnectionIndex == i {
					connectionList.UnselectAll()
				}
				showEditConnectionTabItem(i)
				toggleConnectionsButton.Disable()
			}
			deleteButton := toolbar.Items[1].(*widget.ToolbarAction)
			deleteButton.OnActivated = func() {
				if selectedConnectionIndex == i {
					connectionList.UnselectAll()
				}
				// show confirm dialog
				dialog.ShowConfirm("Delete Connection", "Are you sure you want to delete this connection?", func(b bool) {
					if b {
						deleteConnection(i, connectionList)
						if selectedConnectionIndex == i {
							selectedConnectionIndex = -1
							keyValueTable.UnselectAll()
						}
					}
				}, w)
			}
			toolbar.Refresh()
		},
	)

	connectionList.OnSelected = func(id widget.ListItemID) {
		selectedConnectionIndex = id
		connectToDB(selectedConnectionIndex)
		// hide mainValueSplit
		keyValuesTabItem.Hidden = false
	}

	connectionList.OnUnselected = func(id widget.ListItemID) {
		selectedConnectionIndex = -1
		// show mainValueSplit
		keyValuesTabItem.Hidden = true
		keyValueTable.UnselectAll()
		err := env.Close()
		if err != nil {
			showErrorLog("Error closing LMDB environment: " + err.Error())
			return
		}
	}

	// 主下侧布局：Value 多功能区
	valueLabel := widget.NewLabelWithData(valueLabelString)
	valueLabel.TextStyle = fyne.TextStyle{Bold: true}

	hideButton := widget.NewButtonWithIcon("Hide", theme.ContentRemoveIcon(), func() {
		if selectedKey != "" {
			toggleValue()
			keyValueTable.UnselectAll()
		}
	})

	valueView = widget.NewMultiLineEntry()
	valueView.Wrapping = fyne.TextWrapWord

	updateButton := widget.NewButtonWithIcon("Update", theme.ConfirmIcon(), func() {
		if selectedKey != "" {
			insertOrUpdateKeyValue(selectedKey, valueView.Text)
			keyValueTable.UnselectAll()
		}
	})

	deleteButton := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		if selectedKey != "" {
			deleteKeyValue(selectedKey)
			keyValueTable.UnselectAll()
		}
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		keyValueTable.UnselectAll()
	})

	valueControls := container.NewGridWithColumns(3, updateButton, deleteButton, cancelButton)
	copyLabelButton := widget.NewButtonWithIcon("Copy Key", theme.ContentCopyIcon(), func() {
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(valueLabel.Text[5:])
		showInfoLog("Key copied to clipboard!")
	})
	valuePanel = container.NewBorder(container.NewBorder(nil, nil, container.NewHBox(valueLabel, copyLabelButton), hideButton, nil), valueControls, nil, nil, valueView)
	valuePanel.Hidden = true

	keyValueTable = widget.NewTableWithHeaders(
		func() (int, int) {
			return len(keyValues), 2
		},
		func() fyne.CanvasObject {
			newLabel := widget.NewLabel("")
			newLabel.Alignment = fyne.TextAlignCenter
			return newLabel
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Col == 0 {
				label.SetText(keyValues[i.Row].Key)
			} else { // 1
				label.SetText(keyValues[i.Row].Value)
			}
		},
	)

	keyValueTable.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(keyValues) || id.Col < 0 {
			return
		}
		isHide, err := hideKeyPrefix.Get()
		if err != nil {
			return
		}
		selectedKey = keyValues[id.Row].Key
		if isHide {
			prefix, err := keyPrefix.Get()
			if err != nil {
				return
			}
			selectedKey = prefix + selectedKey
		}
		if err := valueLabelString.Set("Key: " + selectedKey); err != nil {
			return
		}
		refreshValueView(valueView)
		// 如果 Value 多功能区是关闭的，则打开
		if !valuePanelOpen {
			toggleValue()
		}
		valuePanel.Hidden = false
		valuePanelOpen = true
	}

	keyValueTable.OnUnselected = func(id widget.TableCellID) {
		selectedKey = ""
		err := valueLabelString.Set("Key: " + selectedKey)
		if err != nil {
			return
		}
		valueView.SetText("")
		if valuePanelOpen {
			toggleValue()
		}
		valuePanel.Hidden = true
		valuePanelOpen = false
	}

	keyValueTable.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		label := template.(*widget.Label)
		if id.Col == 0 {
			label.SetText("Key")
		} else {
			label.SetText("Value")
		}
	}
	// update column header
	keyValueTable.ShowHeaderColumn = false

	keyPrefixEntry := widget.NewEntryWithData(keyPrefix)
	keyPrefixEntry.SetPlaceHolder("Key prefix filter")
	keyPrefixEntry.OnSubmitted = func(s string) {
		loadKeyValues(s, true)
	}

	// clear key prefix filter button
	clearKeyPrefixButton := widget.NewButtonWithIcon("Clear", theme.CancelIcon(), func() {
		keyPrefixEntry.SetText("")
		loadKeyValues("", true)
	})

	keyPrefixLabel := widget.NewLabel("Key Prefix:")
	keyPrefixLabel.Alignment = fyne.TextAlignLeading

	keyPrefixIcon := widget.NewIcon(theme.SearchIcon())
	keyPrefixLabels := container.NewHBox(keyPrefixIcon, keyPrefixLabel)

	refreshKeysButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		loadKeyValues(keyPrefixEntry.Text, true)
		showInfoLog("Keys refreshed!")
		keyValueTable.UnselectAll()
	})

	unselectKeysButton := widget.NewButtonWithIcon("Unselect", theme.ContentUndoIcon(), func() {
		keyValueTable.UnselectAll()
	})

	newKeyButton := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() {
		showNewKeyValesTabItem()
	})

	err = tabTitle.Set("Key Values")
	if err != nil {
		return
	}
	tabTitleLabel := widget.NewLabelWithData(tabTitle)
	tabTitleLabel.TextStyle = fyne.TextStyle{Bold: true}
	tabTitleLabel.Alignment = fyne.TextAlignCenter

	err = hideKeyPrefix.Set(true)
	if err != nil {
		return
	}

	hideKeyPrefixCheckbox := widget.NewCheckWithData("Hide Key Prefix", hideKeyPrefix)
	hideKeyPrefixCheckbox.OnChanged = func(b bool) {
		err := hideKeyPrefix.Set(b)
		if err != nil {
			return
		}
		if selectedConnectionIndex != -1 {
			loadKeyValues(keyPrefixEntry.Text, false)
		}
	}

	autoRefreshCheckbox := widget.NewCheck("Auto Refresh (5s)", nil)
	autoRefreshCheckbox.Checked = false
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if connectionList.Length() != 0 && selectedConnectionIndex != -1 && autoRefreshCheckbox.Checked {
				// reconnect to db
				loadKeyValues(keyPrefixEntry.Text, true)
				keyValueTable.UnselectAll()

				showInfoLog("Auto Refresh success!")
			}
		}
	}()

	refreshUnselectNewGrid := container.NewGridWithColumns(5, newKeyButton, unselectKeysButton, refreshKeysButton,
		container.NewCenter(hideKeyPrefixCheckbox), container.NewCenter(autoRefreshCheckbox))

	// 添加标题栏左侧的两个按钮
	toggleConnectionsButton = widget.NewButtonWithIcon("Connections", theme.MenuIcon(), toggleConnections)
	switchThemeButton := widget.NewButtonWithIcon("Dark/Light", theme.ViewRefreshIcon(), func() {
		if darkMode {
			a.Settings().SetTheme(lightTheme)
			darkMode = false
		} else {
			a.Settings().SetTheme(darkTheme)
			darkMode = true
		}
		logText.Color = theme.ForegroundColor()
	})

	keyPrefixes := container.NewBorder(nil, nil, keyPrefixLabels, clearKeyPrefixButton, keyPrefixEntry)
	keyValuesControls := container.NewBorder(nil, refreshUnselectNewGrid, nil, nil, keyPrefixes)
	keyValuesList := container.NewBorder(keyValuesControls, nil, nil, nil, keyValueTable)

	connectConnectionButton := widget.NewButtonWithIcon("New Connection", theme.ContentAddIcon(), func() {
		showNewConnectionTabItem()
	})

	connectionsLabel := widget.NewLabel("Connections")
	connectionsLabel.TextStyle = fyne.TextStyle{Bold: true}
	connectionsLabel.Alignment = fyne.TextAlignCenter

	connectionsPanel = container.NewBorder(connectionsLabel, connectConnectionButton, nil, nil, connectionList)

	// 创建主拆分器，将 Key Values列表和 Value 多功能区组合在一起
	keyValuesTabItem = container.NewVSplit(keyValuesList, valuePanel)
	keyValuesTabItem.Offset = 1.0
	keyValuesTabItem.Trailing = container.NewVBox()
	keyValuesTabItem.Hidden = true

	newKeyValesTabItem = initNewKeyValuesTableItem()

	newConnectionTabItem = initNewConnectionTabItem(w)

	editConnectionTabItem = initEditConnectionTabItem(w)

	tabTitles := container.NewBorder(nil, nil, toggleConnectionsButton, switchThemeButton, tabTitleLabel)

	tabView = container.NewStack(keyValuesTabItem, newConnectionTabItem, newKeyValesTabItem, editConnectionTabItem)

	tabContent := container.NewBorder(tabTitles, nil, nil, nil, tabView)

	// 创建主布局，将左侧面板和主拆分器组合在一起
	leftMainSplit = container.NewHSplit(connectionsPanel, tabContent)
	leftMainSplit.Offset = 0.15

	err = glfw.Init()
	if err != nil {
		showErrorLog("Error initializing GLFW: " + err.Error())
		return
	}
	monitor := glfw.GetPrimaryMonitor()
	mode := monitor.GetVideoMode()
	screenWidth := float32(mode.Width)
	screenHeight := float32(mode.Height)
	windowWidth = screenWidth / 4 * 3
	windowWidth = float32(math.Min(float64(windowWidth), 1400))
	windowHeight = screenHeight / 4 * 3
	windowHeight = float32(math.Min(float64(windowHeight), 800))

	// set log labels in the bottom
	logLabel := newLogLabel(logMessage)
	bottomPanel := container.NewBorder(widget.NewSeparator(), nil, nil, logLabel)
	mainContent := container.NewBorder(nil, bottomPanel, nil, nil, leftMainSplit)

	w.SetContent(mainContent)
	w.Resize(fyne.NewSize(windowWidth, windowHeight))
	w.ShowAndRun()
}

func refreshValueView(valueView *widget.Entry) {
	err := env.View(func(txn *lmdb.Txn) error {
		val, err := txn.Get(dbi, []byte(selectedKey))
		if err != nil {
			return err
		}
		var formattedJSON map[string]interface{}
		err = json.Unmarshal(val, &formattedJSON)
		if err != nil {
			valueView.SetText(string(val))
		} else {
			prettyJSON, _ := json.MarshalIndent(formattedJSON, "", "  ")
			valueView.SetText(string(prettyJSON))
		}
		return nil
	})
	if err != nil {
		showErrorLog("Error fetching value: " + err.Error())
	}
}

func deleteConnection(connectionIndex int, connectionList *widget.List) {
	config.Config.Connections = append(config.Config.Connections[:connectionIndex], config.Config.Connections[connectionIndex+1:]...)
	err := config.SaveConfig()
	if err != nil {
		showErrorLog("Error saving config: " + err.Error())
	}
	connectionList.Refresh()
}

func connectToDB(connectionIndex int) {
	if len(config.Config.Connections) == 0 {
		showErrorLog("No database path configured")
		return
	}
	connection := config.Config.Connections[connectionIndex]

	var err error
	env, err = lmdb.NewEnv()
	if err != nil {
		showErrorLog("Error creating LMDB environment: " + err.Error())
		return
	}

	err = env.SetMapSize(1 << 30 * connection.MapSize)
	if err != nil {
		showErrorLog("Error setting LMDB map size: " + err.Error())
		return
	}

	err = env.SetMaxDBs(0)
	if err != nil {
		showErrorLog("Error setting LMDB max DBs: " + err.Error())
		return
	}
	err = env.Open(connection.DatabasePath, 0, 0664)
	if err != nil {
		showErrorLog("Error opening LMDB database: " + err.Error())
		return
	}
	err = env.Update(func(txn *lmdb.Txn) (err error) {
		dbi, err = txn.OpenRoot(0)
		return err
	})
	if err != nil {
		showErrorLog("Error opening LMDB root: " + err.Error())
		return
	}
	showInfoLog("Database connected")

	loadKeyValues("", false)
}

func loadKeyValues(keyPrefix string, reconnectDB bool) {
	if reconnectDB {
		connectToDB(selectedConnectionIndex)
	}
	err := env.View(func(txn *lmdb.Txn) error {
		scanner := lmdbscan.New(txn, dbi)

		// 设置扫描器的起始位置
		if keyPrefix != "" {
			scanner.Set([]byte(keyPrefix), nil, lmdb.SetRange)
		}

		keyValues = make([]KeyValue, 0)
		for scanner.Scan() {
			key := scanner.Key()
			val := scanner.Val()

			// 检查键前缀
			if keyPrefix == "" || (len(key) >= len(keyPrefix) && string(key[:len(keyPrefix)]) == keyPrefix) {
				displayVal := string(val)
				//if len(displayVal) > 50 {
				//	displayVal = displayVal[:50] + "..."
				//}

				displayKey := string(key)
				hidePrefix, err := hideKeyPrefix.Get()
				if err != nil {
					return err
				}
				if hidePrefix {
					displayKey = displayKey[len(keyPrefix):]
				}
				keyValues = append(keyValues, KeyValue{Key: displayKey, Value: strings.ReplaceAll(displayVal, "\n", " ")})
			} else if keyPrefix != "" && string(key) > keyPrefix {
				// 如果当前键大于前缀，结束扫描
				break
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		keyValueTable.Refresh()
		adaptiveColumnWidths()

		return nil
	})
	if err != nil {
		showErrorLog("Error loading keys: " + err.Error())
	}
}

func insertOrUpdateKeyValue(key, value string) {
	err := env.Update(func(txn *lmdb.Txn) error {
		err := txn.Put(dbi, []byte(key), []byte(value), 0)
		return err
	})
	if err != nil {
		showErrorLog("Error insert/update key-value: " + err.Error())
		return
	}
	showInfoLog("Key-Value inserted/updated")
	prefix, err := keyPrefix.Get()
	if err != nil {
		return
	}
	loadKeyValues(prefix, false)
}

func deleteKeyValue(key string) {
	err := env.Update(func(txn *lmdb.Txn) error {
		err := txn.Del(dbi, []byte(key), nil)
		return err
	})
	if err != nil {
		showErrorLog("Error deleting key-value: " + err.Error())
		return
	}
	showInfoLog("Key-Value deleted")

	prefix, err := keyPrefix.Get()
	if err != nil {
		return
	}

	loadKeyValues(prefix, false)
}

func initEditConnectionTabItem(w fyne.Window) *fyne.Container {
	editConnectionNameLabel := widget.NewLabel("Connection Name:")
	editConnectionNameLabel.TextStyle = fyne.TextStyle{Monospace: true}
	editConnectionNameEntry = widget.NewEntry()
	editConnectionPathLabel := widget.NewLabel("Database  Path :")
	editConnectionPathLabel.TextStyle = fyne.TextStyle{Monospace: true}
	editConnectionPathEntry = widget.NewEntry()
	editConnectionMapSizeLabel := widget.NewLabel("Map  Size  (GB) :")
	editConnectionMapSizeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	editConnectionMapSizeEntry = widget.NewEntry()

	saveButton := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		if editConnectionNameEntry.Text == "" {
			showErrorLog("Connection name cannot be empty")
			return
		}
		if editConnectionPathEntry.Text == "" {
			showErrorLog("Database path cannot be empty")
			return
		}
		if editConnectionMapSizeEntry.Text == "" {
			showErrorLog("Map size cannot be empty")
			return
		}
		// check map size is a non negative integer
		if !isPositiveInteger(editConnectionMapSizeEntry.Text) {
			showErrorLog("Map size must be a non-negative integer")
			return
		}

		// try to open the database to check if it exists
		envTest, err := lmdb.NewEnv()
		if err != nil {
			showErrorLog("Error creating LMDB environment: " + err.Error())
			return
		}
		err = envTest.Open(editConnectionPathEntry.Text, 0, 0664)
		if err != nil {
			showErrorLog("Error opening LMDB database: " + err.Error())
			return
		}
		err = envTest.Close()
		if err != nil {
			showErrorLog("Error closing LMDB environment: " + err.Error())
			return
		}

		config.Config.Connections[editConnectionIndex].Name = editConnectionNameEntry.Text
		config.Config.Connections[editConnectionIndex].DatabasePath = editConnectionPathEntry.Text
		// convert map size to int64
		mapSize, err := strconv.ParseInt(editConnectionMapSizeEntry.Text, 10, 64)
		if err != nil {
			showErrorLog("Error converting map size to int64: " + err.Error())
			return
		}

		// try to open the database to check if it exists
		envTest, err = lmdb.NewEnv()
		if err != nil {
			showErrorLog("Error creating LMDB environment: " + err.Error())
			return
		}
		err = envTest.Open(editConnectionPathEntry.Text, 0, 0664)
		if err != nil {
			showErrorLog("Error opening LMDB database: " + err.Error())
			return
		}
		err = envTest.Close()
		if err != nil {
			showErrorLog("Error closing LMDB environment: " + err.Error())
			return
		}

		config.Config.Connections[editConnectionIndex].MapSize = mapSize
		err = config.SaveConfig()
		if err != nil {
			showErrorLog("Error saving config: " + err.Error())
		}
		connectionList.Refresh()

		err = tabTitle.Set("Key Values")
		if err != nil {
			return
		}
		editConnectionTabItem.Hide()

		// open the connections panel
		toggleConnections()
		toggleConnectionsButton.Enable()
	})

	browseButton := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFolderOpen(func(file fyne.ListableURI, err error) {
			if file != nil {
				path := file.Path()
				if path[len(path)-1] != '/' {
					path += "/"
				}
				editConnectionPathEntry.SetText(path)
			}
		}, w)
		fd.Resize(fyne.NewSize(windowWidth, windowHeight))
		fd.Show()
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		err := tabTitle.Set("Key Values")
		if err != nil {
			return
		}
		editConnectionTabItem.Hide()
		// open the connections panel
		toggleConnections()
		toggleConnectionsButton.Enable()
	})

	// 确保输入框尽可能大
	border := container.NewVBox(
		container.NewBorder(nil, nil, editConnectionNameLabel, nil, editConnectionNameEntry),
		container.NewBorder(nil, nil, editConnectionPathLabel, browseButton, editConnectionPathEntry),
		container.NewBorder(nil, nil, editConnectionMapSizeLabel, nil, editConnectionMapSizeEntry),
		container.NewGridWithColumns(2, saveButton, cancelButton),
	)
	border.Hide()
	return border
}

func initNewConnectionTabItem(w fyne.Window) *fyne.Container {
	nameLabel := widget.NewLabel("Connection Name:")
	nameLabel.TextStyle = fyne.TextStyle{Monospace: true}
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter connection name")
	entryLabel := widget.NewLabel("Database  Path :")
	entryLabel.TextStyle = fyne.TextStyle{Monospace: true}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter database path or use the button to browse")
	mapSizeLabel := widget.NewLabel("Map  Size  (GB) :")
	mapSizeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	mapSizeEntry := widget.NewEntry()
	mapSizeEntry.SetText("1")

	browseButton := widget.NewButtonWithIcon("Browse", theme.FolderNewIcon(), func() {
		fd := dialog.NewFolderOpen(func(file fyne.ListableURI, err error) {
			if file != nil {
				path := file.Path()
				if path[len(path)-1] != '/' {
					path += "/"
				}
				entry.SetText(path)
			}
		}, w)
		fd.Resize(fyne.NewSize(windowWidth, windowHeight))
		fd.Show()
	})

	saveButton := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		if nameEntry.Text == "" {
			showErrorLog("Connection name cannot be empty")
			return
		}
		if entry.Text == "" {
			showErrorLog("Database path cannot be empty")
			return
		}
		if mapSizeEntry.Text == "" {
			showErrorLog("Map size cannot be empty")
			return
		}
		// check map size is a non negative integer
		if !isPositiveInteger(mapSizeEntry.Text) {
			showErrorLog("Map size must be a non-negative integer")
			return
		}
		// try to open the database to check if it exists
		envTest, err := lmdb.NewEnv()
		if err != nil {
			showErrorLog("Error creating LMDB environment: " + err.Error())
			return
		}
		err = envTest.Open(entry.Text, 0, 0664)
		if err != nil {
			showErrorLog("Error opening LMDB database: " + err.Error())
			return
		}
		err = envTest.Close()
		if err != nil {
			showErrorLog("Error closing LMDB environment: " + err.Error())
			return
		}

		mapSize, err := strconv.ParseInt(mapSizeEntry.Text, 10, 64)
		config.Config.Connections = append(config.Config.Connections, config.ConnectionConfig{
			Name:         nameEntry.Text,
			DatabasePath: entry.Text,
			MapSize:      mapSize,
		})
		err = config.SaveConfig()
		if err != nil {
			showErrorLog("Error saving config: " + err.Error())
		}
		connectionList.Refresh()

		nameEntry.SetText("")
		entry.SetText("")
		mapSizeEntry.SetText("1")
		err = tabTitle.Set("Key Values")
		if err != nil {
			return
		}
		newConnectionTabItem.Hide()
	})

	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		nameEntry.SetText("")
		entry.SetText("")
		mapSizeEntry.SetText("1")
		err := tabTitle.Set("Key Values")
		if err != nil {
			return
		}
		newConnectionTabItem.Hide()
	})

	// 确保输入框尽可能大
	border := container.NewVBox(
		container.NewBorder(nil, nil, nameLabel, nil, nameEntry),
		container.NewBorder(nil, nil, entryLabel, browseButton, entry),
		container.NewBorder(nil, nil, mapSizeLabel, nil, mapSizeEntry),
		container.NewGridWithColumns(2, saveButton, cancelButton),
	)
	border.Hide()
	return border
}

func initNewKeyValuesTableItem() *fyne.Container {
	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder("Enter key")
	valueEntry := widget.NewMultiLineEntry()
	valueEntry.SetPlaceHolder("Enter value")
	valueEntry.Wrapping = fyne.TextWrapWord

	saveButton := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		insertOrUpdateKeyValue(keyEntry.Text, valueEntry.Text)
		keyEntry.SetText("")
		valueEntry.SetText("")
		showKeyValesTabItem()
	})
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		keyEntry.SetText("")
		valueEntry.SetText("")
		showKeyValesTabItem()
	})

	// 确保输入框尽可能大
	border := container.NewBorder(
		keyEntry, // top
		container.NewGridWithColumns(2, saveButton, cancelButton), // bottom
		nil,        // left
		nil,        // right
		valueEntry, // center
	)
	border.Hide()
	return border
}

func showNewConnectionTabItem() {
	err := tabTitle.Set("New Connection")
	if err != nil {
		return
	}
	newConnectionTabItem.Show()
	editConnectionTabItem.Hide()
	newKeyValesTabItem.Hide()
	keyValuesTabItem.Hide()
}

func showEditConnectionTabItem(i int) {
	err := tabTitle.Set("Edit Connection")
	if err != nil {
		return
	}
	editConnectionIndex = i

	connection := config.Config.Connections[editConnectionIndex]
	editConnectionNameEntry.SetText(connection.Name)
	editConnectionPathEntry.SetText(connection.DatabasePath)
	editConnectionMapSizeEntry.SetText(strconv.FormatInt(connection.MapSize, 10))

	newConnectionTabItem.Hide()
	editConnectionTabItem.Show()
	newKeyValesTabItem.Hide()
	keyValuesTabItem.Hide()

	// close the connections panel
	toggleConnections()
}

func showKeyValesTabItem() {
	newKeyValesTabItem.Hide()
	editConnectionTabItem.Hide()
	newConnectionTabItem.Hide()
	keyValuesTabItem.Show()
	err := tabTitle.Set("Key Values")
	if err != nil {
		return
	}
}

func showNewKeyValesTabItem() {
	err := tabTitle.Set("New Key-Value")
	if err != nil {
		return
	}
	newKeyValesTabItem.Show()
	editConnectionTabItem.Hide()
	newConnectionTabItem.Hide()
	keyValuesTabItem.Hide()
}

func toggleConnections() {
	if connectionPanelOpen {
		leftMainSplit.Leading = container.NewVBox()
		leftMainSplit.Offset = 0.0
		connectionPanelOpen = false
	} else {
		leftMainSplit.Leading = connectionsPanel
		leftMainSplit.Offset = 0.15
		connectionPanelOpen = true
	}
	leftMainSplit.Refresh()
	adaptiveColumnWidths()
}

func toggleValue() {
	if valuePanelOpen {
		keyValuesTabItem.Trailing = container.NewVBox()
		if keyValuesTabItem.Offset != 1.0 && valueSplitOffset != keyValuesTabItem.Offset {
			valueSplitOffset = keyValuesTabItem.Offset
		}
		keyValuesTabItem.Offset = 1.0
	} else {
		keyValuesTabItem.Trailing = valuePanel
		keyValuesTabItem.Offset = valueSplitOffset
	}
	keyValuesTabItem.Refresh()
}

func showInfoLog(message string) {
	localTimeStr := time.Now().Format("2006-01-02 15:04:05")
	message = localTimeStr + "｜INFO｜" + message
	err := logMessage.Set(message)
	if err != nil {
		return
	}

	go func() {
		time.Sleep(5 * time.Second)

		if oldMessage, _ := logMessage.Get(); oldMessage == message {
			err := logMessage.Set("")
			if err != nil {
				return
			}
		}
	}()
}

func showErrorLog(message string) {
	localTimeStr := time.Now().Format("2006-01-02 15:04:05")
	message = localTimeStr + "｜ERROR｜" + message
	err := logMessage.Set(message)
	if err != nil {
		return
	}

	go func() {
		time.Sleep(5 * time.Second)

		if oldMessage, _ := logMessage.Get(); oldMessage == message {
			err := logMessage.Set("")
			if err != nil {
				return
			}
		}
	}()
}

func newLogLabel(data binding.String) *fyne.Container {
	logText = canvas.NewText("", theme.ForegroundColor())
	logText.TextSize = 14

	// 绑定数据到文本内容
	data.AddListener(binding.NewDataListener(func() {
		value, _ := data.Get()

		logText.Text = value + "  "
		logText.Refresh()
	}))
	return container.NewVBox(logText)
}

func adaptiveColumnWidths() {
	if len(keyValues) == 0 || len(keyValues) > 300 {
		return
	}

	// 预设列的最大宽度
	maxKeyWidth := float32(300)
	maxValueWidth := float32(300)

	// 遍历所有的键值对，计算列的最大宽度
	for _, keyValue := range keyValues {
		keyWidth := fyne.MeasureText(keyValue.Key, theme.TextSize(), fyne.TextStyle{}).Width
		if keyWidth > maxKeyWidth {
			maxKeyWidth = keyWidth
		}
	}

	// 增加一些额外的空间
	maxKeyWidth += 10

	// 设置表格列的宽度
	keyValueTable.SetColumnWidth(0, maxKeyWidth)

	// 计算剩余的宽度并更新值为前缀那么多字
	remainingWidth := keyValuesTabItem.Size().Width - maxKeyWidth
	minValueWidth := fyne.MeasureText("W", theme.TextSize(), fyne.TextStyle{}).Width * 30 // 计算30个字符的宽度

	if remainingWidth < minValueWidth {
		remainingWidth = minValueWidth
	}

	for i, keyValue := range keyValues {
		value := keyValue.Value
		prefixValue := truncateToFit(value, remainingWidth)
		keyValues[i].Value = prefixValue
	}

	// 更新值列的宽度
	maxValueWidth = remainingWidth
	keyValueTable.SetColumnWidth(1, maxValueWidth)
}

// truncateToFit truncates the given string to fit within the specified width
func truncateToFit(value string, width float32) string {
	var truncatedValue string
	space := fyne.MeasureText(" ", theme.TextSize(), fyne.TextStyle{}).Width
	for i := 1; i <= len(value); i++ {
		partialValue := value[:i]
		partialWidth := fyne.MeasureText(partialValue, theme.TextSize(), fyne.TextStyle{}).Width
		if partialWidth+space > width {
			truncatedValue = value[:i-1]
			break
		}
		truncatedValue = partialValue
	}
	return truncatedValue
}

func isPositiveInteger(s string) bool {
	// 尝试将字符串转换为整数
	n, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	// 检查整数是否为正数
	return n > 0
}
