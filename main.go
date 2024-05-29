package main

import (
	"encoding/json"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/bmatsuo/lmdb-go/lmdb"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/zshimonz/lmdb-gui-client/config"
	"github.com/zshimonz/lmdb-gui-client/theme"
)

var env *lmdb.Env
var dbi lmdb.DBI
var keyValues []KeyValue
var selectedKey string
var windowWidth float32
var windowHeight float32
var connectionList *widget.List
var keyList *widget.List
var valueView *widget.Entry
var selectedConnectionIndex = -1

var connectionsPanel *fyne.Container
var valuePanel *fyne.Container
var leftMainSplit *container.Split
var mainValueSplit *container.Split

type KeyValue struct {
	Key   string
	Value string
}

func main() {
	a := app.New()

	// 加载嵌入的字体
	customTheme := &theme.MyTheme{}
	a.Settings().SetTheme(customTheme)

	w := a.NewWindow("LMDB GUI Client")

	err := config.LoadConfig()
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error loading config: "+err.Error())
	}

	// 左侧布局：Connection 列表
	connectionList = widget.NewList(
		func() int { return len(config.Config.Connections) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			editButton := widget.NewButton("Edit", func() {})
			deleteButton := widget.NewButton("Delete", func() {})
			return container.NewHBox(label, editButton, deleteButton)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*fyne.Container).Objects[0].(*widget.Label)
			label.SetText(config.Config.Connections[i].Name)

			editButton := o.(*fyne.Container).Objects[1].(*widget.Button)
			editButton.OnTapped = func() {
				showEditConnectionWindow(a, w, i, connectionList)
			}

			deleteButton := o.(*fyne.Container).Objects[2].(*widget.Button)
			deleteButton.OnTapped = func() {
				deleteConnection(a, w, i, connectionList)
			}
		},
	)

	connectionList.OnSelected = func(id widget.ListItemID) {
		selectedConnectionIndex = id
		connectToDB(a, w, selectedConnectionIndex, keyList, valueView)
		// hide mainValueSplit
		mainValueSplit.Hidden = false
	}

	connectionList.OnUnselected = func(id widget.ListItemID) {
		selectedConnectionIndex = -1
		// show mainValueSplit
		mainValueSplit.Hidden = true
	}

	// 主下侧布局：Value 多功能区
	valueLabel := widget.NewLabel("Value")

	valueView = widget.NewMultiLineEntry()
	valueView.Wrapping = fyne.TextWrapWord

	updateButton := widget.NewButton("Update", func() {
		if selectedKey != "" {
			insertOrUpdateKeyValue(a, w, selectedKey, valueView.Text, keyList)
			toggleValue()
			valueView.Hidden = true
		}
	})

	deleteButton := widget.NewButton("Delete", func() {
		if selectedKey != "" {
			deleteKeyValue(a, w, selectedKey, keyList)
			toggleValue()
			valueView.Hidden = true
		}
	})

	cancelButton := widget.NewButton("Cancel", func() {
		toggleValue()
		valueView.Hidden = true
	})

	valueControls := container.NewGridWithColumns(3, updateButton, deleteButton, cancelButton)
	valuePanel = container.NewBorder(valueLabel, valueControls, nil, nil, valueView)
	valuePanel.Hidden = true

	// 中间布局：Key 列表
	keyList = widget.NewList(
		func() int { return len(keyValues) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.Alignment = fyne.TextAlignLeading
			displayText := keyValues[i].Key
			if len(keyValues[i].Value) > 0 {
				displayText += ": " + strings.ReplaceAll(keyValues[i].Value, "\n", " ")
			}
			label.SetText(displayText)
		},
	)

	keyList.OnSelected = func(id widget.ListItemID) {
		selectedKey = keyValues[id].Key
		refreshValueView(a, w, valueView)
		// 如果 Value 多功能区是关闭的，则打开
		if mainValueSplit.Offset == 1.0 {
			toggleValue()
		}
		valuePanel.Hidden = false
	}

	keyList.OnUnselected = func(id widget.ListItemID) {
		selectedKey = ""
		valueView.SetText("")
		if mainValueSplit.Offset == 0.6 {
			toggleValue()
		}
		valuePanel.Hidden = true
	}

	keyPrefixEntry := widget.NewEntry()
	keyPrefixEntry.SetPlaceHolder("Key prefix filter")
	keyPrefixEntry.OnSubmitted = func(s string) {
		loadKeys(a, w, keyList, s)
	}

	keyPrefixLabel := widget.NewLabel("Key Prefix:")
	keyPrefixLabel.Alignment = fyne.TextAlignLeading

	keyPrefixBroder := container.NewBorder(nil, nil, keyPrefixLabel, nil, keyPrefixEntry)

	refreshKeysButton := widget.NewButton("Refresh Keys", func() {
		loadKeys(a, w, keyList, keyPrefixEntry.Text)
	})

	newKeyButton := widget.NewButton("New Key", func() {
		showNewKeyWindow(a, w, keyList)
	})

	keysLabel := widget.NewLabel("Key Values")
	keysLabel.TextStyle = fyne.TextStyle{Bold: true}
	keysLabel.Alignment = fyne.TextAlignCenter

	autoRefreshCheckbox := widget.NewCheck("Auto Refresh", nil)
	autoRefreshCheckbox.Checked = false
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if connectionList.Length() != 0 && selectedConnectionIndex != -1 && autoRefreshCheckbox.Checked {
				// reconnect to db
				connectToDB(a, w, selectedConnectionIndex, keyList, valueView)
				loadKeys(a, w, keyList, keyPrefixEntry.Text)
				keyList.UnselectAll()

				showLogPopup(a, w, "INFO", "Auto Refresh success!")
			}
		}
	}()

	refreshNewGrid := container.NewGridWithColumns(2, refreshKeysButton, newKeyButton)

	// 添加标题栏左侧的两个按钮
	toggleConnectionsButton := widget.NewButton("Toggle Connections", toggleConnections)
	toggleValueButton := widget.NewButton("Toggle Value", toggleValue)
	toolbar := container.NewHBox(toggleConnectionsButton, toggleValueButton)

	keysControls := container.NewBorder(nil, nil, toolbar, autoRefreshCheckbox, keysLabel)
	keyControls := container.NewVBox(keysControls, keyPrefixBroder, refreshNewGrid)
	keyListContainer := container.NewBorder(keyControls, nil, nil, nil, keyList)

	connectionsPanel = container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		connectionList,
	)

	connectButton := widget.NewButton("New Connection", func() {
		showConnectWindow(a, w, connectionList)
	})
	connectionsPanel = container.NewBorder(connectButton, nil, nil, nil, connectionsPanel)

	// 创建主拆分器，将 Key Values列表和 Value 多功能区组合在一起
	mainValueSplit = container.NewVSplit(keyListContainer, valuePanel)
	mainValueSplit.Offset = 1.0
	mainValueSplit.Trailing = container.NewVBox()
	mainValueSplit.Hidden = true

	// 创建主布局，将左侧面板和主拆分器组合在一起
	leftMainSplit = container.NewHSplit(connectionsPanel, mainValueSplit)
	leftMainSplit.Offset = 0.15

	err = glfw.Init()
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error initializing GLFW: "+err.Error())
		return
	}
	monitor := glfw.GetPrimaryMonitor()
	mode := monitor.GetVideoMode()
	screenWidth := float32(mode.Width)
	screenHeight := float32(mode.Height)
	windowWidth = screenWidth / 4 * 3
	windowHeight = screenHeight / 4 * 3
	w.SetContent(leftMainSplit)
	w.Resize(fyne.NewSize(windowWidth, windowHeight))
	w.ShowAndRun()
}

func refreshValueView(a fyne.App, w fyne.Window, valueView *widget.Entry) {
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
		showLogPopup(a, w, "ERROR", "Error fetching value: "+err.Error())
	}
}

func deleteConnection(a fyne.App, w fyne.Window, connectionIndex int, connectionList *widget.List) {
	config.Config.Connections = append(config.Config.Connections[:connectionIndex], config.Config.Connections[connectionIndex+1:]...)
	err := config.SaveConfig()
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error saving config: "+err.Error())
	}
	connectionList.Refresh()
}

func connectToDB(a fyne.App, w fyne.Window, connectionIndex int, keyList *widget.List, valueView *widget.Entry) {
	if len(config.Config.Connections) == 0 {
		showLogPopup(a, w, "ERROR", "No database path configured")
		return
	}
	connection := config.Config.Connections[connectionIndex]

	var err error
	env, err = lmdb.NewEnv()
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error creating LMDB environment: "+err.Error())
		return
	}

	err = env.SetMapSize(1 << 30 * 100)
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error setting LMDB map size: "+err.Error())
		return
	}

	err = env.SetMaxDBs(1)
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error setting LMDB max DBs: "+err.Error())
		return
	}
	err = env.Open(connection.DatabasePath, 0, 0664)
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error opening LMDB database: "+err.Error())
		return
	}
	err = env.Update(func(txn *lmdb.Txn) (err error) {
		dbi, err = txn.OpenRoot(0)
		return err
	})
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error opening LMDB root: "+err.Error())
		return
	}
	showLogPopup(a, w, "INFO", "Database connected")

	loadKeys(a, w, keyList, "")
}

func loadKeys(a fyne.App, w fyne.Window, keyList *widget.List, keyPrefix string) {
	err := env.View(func(txn *lmdb.Txn) error {
		cur, err := txn.OpenCursor(dbi)
		if err != nil {
			return err
		}
		defer cur.Close()

		keyValues = nil
		for {
			key, val, err := cur.Get(nil, nil, lmdb.Next)
			if err != nil {
				break
			}
			if keyPrefix == "" || len(key) >= len(keyPrefix) && string(key[:len(keyPrefix)]) == keyPrefix {
				displayVal := string(val)
				if len(displayVal) > 50 {
					displayVal = displayVal[:50] + "..."
				}
				keyValues = append(keyValues, KeyValue{Key: string(key), Value: strings.ReplaceAll(displayVal, "\n", " ")})
			}
		}

		keyList.Refresh()

		return nil
	})
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error loading keys: "+err.Error())
	}
}

func insertOrUpdateKeyValue(a fyne.App, w fyne.Window, key, value string, keyList *widget.List) {
	err := env.Update(func(txn *lmdb.Txn) error {
		err := txn.Put(dbi, []byte(key), []byte(value), 0)
		return err
	})
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error insert/update key-value: "+err.Error())
		return
	}
	showLogPopup(a, w, "INFO", "Key-Value inserted/updated")

	loadKeys(a, w, keyList, "")
}

func deleteKeyValue(a fyne.App, w fyne.Window, key string, keyList *widget.List) {
	err := env.Update(func(txn *lmdb.Txn) error {
		err := txn.Del(dbi, []byte(key), nil)
		return err
	})
	if err != nil {
		showLogPopup(a, w, "ERROR", "Error deleting key-value: "+err.Error())
		return
	}
	showLogPopup(a, w, "INFO", "Key-Value deleted")

	loadKeys(a, w, keyList, "")
}

func showEditConnectionWindow(a fyne.App, w fyne.Window, connectionIndex int, connectionList *widget.List) {
	connection := config.Config.Connections[connectionIndex]
	editWindow := a.NewWindow("Edit Connection")
	nameEntry := widget.NewEntry()
	nameEntry.SetText(connection.Name)
	entry := widget.NewEntry()
	entry.SetText(connection.DatabasePath)

	saveButton := widget.NewButton("Save", func() {
		config.Config.Connections[connectionIndex].Name = nameEntry.Text
		config.Config.Connections[connectionIndex].DatabasePath = entry.Text
		err := config.SaveConfig()
		if err != nil {
			showLogPopup(a, w, "ERROR", "Error saving config: "+err.Error())
		}
		editWindow.Close()
		connectionList.Refresh()
	})

	browseButton := widget.NewButton("Browse", func() {
		fd := dialog.NewFolderOpen(func(file fyne.ListableURI, err error) {
			if file != nil {
				path := file.Path()
				if path[len(path)-1] != '/' {
					path += "/"
				}
				entry.SetText(path)
			}
		}, editWindow)
		fd.Show()
	})

	// 确保输入框尽可能大
	content := container.NewBorder(
		nameEntry, // top
		container.NewVBox(browseButton, saveButton), // bottom
		nil,   // left
		nil,   // right
		entry, // center
	)

	editWindow.SetContent(content)
	editWindow.Resize(fyne.NewSize(windowWidth/2, windowHeight/3))
	editWindow.Show()
}

func showConnectWindow(a fyne.App, w fyne.Window, connectionList *widget.List) {
	connectWindow := a.NewWindow("New Connection")
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter connection name")
	entry := widget.NewMultiLineEntry()
	entry.SetPlaceHolder("Enter database path or use the button to browse")
	entry.Wrapping = fyne.TextWrapWord

	browseButton := widget.NewButton("Browse", func() {
		fd := dialog.NewFolderOpen(func(file fyne.ListableURI, err error) {
			if file != nil {
				path := file.Path()
				if path[len(path)-1] != '/' {
					path += "/"
				}
				entry.SetText(path)
			}
		}, connectWindow)
		fd.Show()
	})

	saveButton := widget.NewButton("Save", func() {
		config.Config.Connections = append(config.Config.Connections, config.ConnectionConfig{
			Name:         nameEntry.Text,
			DatabasePath: entry.Text,
		})
		err := config.SaveConfig()
		if err != nil {
			showLogPopup(a, w, "ERROR", "Error saving config: "+err.Error())
		}
		connectWindow.Close()
		connectionList.Refresh()
	})

	// 确保输入框尽可能大
	content := container.NewBorder(
		nameEntry, // top
		container.NewVBox(browseButton, saveButton), // bottom
		nil,   // left
		nil,   // right
		entry, // center
	)

	connectWindow.SetContent(content)
	connectWindow.Resize(fyne.NewSize(windowWidth/2, windowHeight/3))
	connectWindow.Show()
}

func showNewKeyWindow(a fyne.App, w fyne.Window, keyList *widget.List) {
	newKeyWindow := a.NewWindow("New Key-Value")
	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder("Enter key")
	valueEntry := widget.NewMultiLineEntry()
	valueEntry.SetPlaceHolder("Enter value")
	valueEntry.Wrapping = fyne.TextWrapWord

	saveButton := widget.NewButton("Save", func() {
		insertOrUpdateKeyValue(a, w, keyEntry.Text, valueEntry.Text, keyList)
		newKeyWindow.Close()
	})

	// 确保输入框尽可能大
	content := container.NewBorder(
		keyEntry,   // top
		saveButton, // bottom
		nil,        // left
		nil,        // right
		valueEntry, // center
	)

	newKeyWindow.SetContent(content)
	newKeyWindow.Resize(fyne.NewSize(windowWidth/2, windowHeight/2))
	newKeyWindow.Show()
}

func toggleConnections() {
	if leftMainSplit.Offset == 0.15 {
		leftMainSplit.Leading = container.NewVBox()
		leftMainSplit.Offset = 0.0
	} else {
		leftMainSplit.Leading = connectionsPanel
		leftMainSplit.Offset = 0.15
	}
	leftMainSplit.Refresh()
}

func toggleValue() {
	if mainValueSplit.Offset == 0.6 {
		mainValueSplit.Trailing = container.NewVBox()
		mainValueSplit.Offset = 1.0
	} else {
		mainValueSplit.Trailing = valuePanel
		mainValueSplit.Offset = 0.6
	}
	mainValueSplit.Refresh()
}

func showLogPopup(a fyne.App, w fyne.Window, logLevel, message string) {
	formattedMessage := "[" + logLevel + "] " + message
	label := widget.NewLabel(formattedMessage)
	content := container.NewVBox(label)
	popup := widget.NewPopUp(content, w.Canvas())

	// set popup at the top mid of the mainValueSplit
	width := mainValueSplit.Size().Width + leftMainSplit.Leading.Size().Width

	popup.Move(fyne.NewPos(width/2, 0))

	popup.Show()

	// 自动隐藏弹出窗口
	go func() {
		time.Sleep(3 * time.Second)
		popup.Hide()
	}()
}
