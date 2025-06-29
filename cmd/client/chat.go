package main

import (
	"fmt"
	"hillside/internal/models"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)


type ChatScreen struct {
	*UI
	layout        *tview.Flex
	GetServerName func() string
    GetServerId   func() string
    GetRoomName    func() string
	roomList	 *tview.List
	roomPane	 *tview.Flex
    roomWrapper  *tview.Flex
	chatView      *tview.Flex
    chatSection  *tview.List
    createBtn  *tview.Button
    modalForm *tview.Form
	rooms 	   []models.RoomMeta
	noRoomView    *tview.TextView
    OnJoinRoom func (roomID string, pass string) error
    sendMessage func (message string) error
    OnCreateRoom func (req models.CreateRoomRequest) (string, error)
    msgInput *tview.TextArea
    sendButton *tview.Button
    joinForm *tview.Form
    selectedRoom models.RoomMeta
}


func (c *ChatScreen) NewChatScreen() {
	c.layout = tview.NewFlex()
	c.layout.SetDirection(tview.FlexColumn).
		SetBorder(false)

	c.roomList = tview.NewList()
    c.roomList.SetSelectedBackgroundColor(c.UI.Theme.GetColor("background-light"))
    c.roomList.SetSelectedTextColor(c.UI.Theme.GetColor("primary")).
        SetHighlightFullLine(true)

	c.roomList.
        SetTitleColor(c.Theme.GetColor("primary")).
        SetBackgroundColor(c.Theme.GetColor("background"))

    c.createBtn = tview.NewButton("Create Room")
    c.createBtn.SetSelectedFunc(c.showCreateRoomForm).
        SetLabelColor(c.Theme.GetColor("button-text")).
        SetBackgroundColor(c.Theme.GetColor("button-active"))


	c.roomPane = tview.NewFlex()
    c.roomPane.AddItem(c.roomList, 0, 1, false)

    c.roomPane.SetDirection(tview.FlexRow)


    c.roomWrapper = tview.NewFlex()
    c.roomWrapper.SetDirection(tview.FlexRow)
    c.roomWrapper.AddItem(c.roomPane, 0, 1, false).
        AddItem(c.createBtn, 1, 0, false)
    c.roomWrapper.SetBorder(true).
        SetTitle(fmt.Sprintf("[ %s ]", c.GetServerName())).
        SetTitleColor(c.Theme.GetColor("primary")).
        SetBorderColor(c.Theme.GetColor("border")).
        SetBackgroundColor(c.Theme.GetColor("background")).
        SetBorderPadding(2,2,2,2)




    c.msgInput = tview.NewTextArea().
        SetPlaceholder("Type your message here...").
        SetPlaceholderStyle(
            tcell.StyleDefault.Background(c.Theme.GetColor("background")).
            Foreground(c.Theme.GetColor("foreground-dark"))).
        SetTextStyle(tcell.StyleDefault.Background(c.Theme.GetColor("background")).
            Foreground(c.Theme.GetColor("foreground")))
    
    c.msgInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        if event.Key() == tcell.KeyEnter {
            message := c.msgInput.GetText()
            if message != "" {
                err := c.sendMessage(message)
                if err != nil {
                    c.UI.ShowError("Send message failed", err.Error(), "OK", 0, nil)
                    return nil
                }
                c.msgInput.SetText("",false)
            }
            return nil
        } else if event.Key() == tcell.KeyEscape {
            c.UI.App.SetFocus(c.chatSection) // Focus back to chat section on Escape
            return nil
        }
        return event
    })

    c.msgInput.SetWordWrap(true).SetWrap(true)
    c.msgInput.SetBorder(true).
        SetBorderColor(c.Theme.GetColor("foreground"))


    c.chatSection = tview.NewList() //where the messages will be displayed
    c.chatSection.SetSelectedBackgroundColor(c.UI.Theme.GetColor("background-light"))
    

    c.chatView = tview.NewFlex()
    c.chatView.SetDirection(tview.FlexRow)
    c.chatView.AddItem(c.chatSection, 0, 1, false).
        AddItem(c.msgInput, 5, 0, true)


    c.chatSection.SetBorder(true).
        SetTitle(fmt.Sprintf("[ %s ]", c.GetRoomName())).
        SetTitleColor(c.Theme.GetColor("primary")).
        SetBorderColor(c.Theme.GetColor("border")).
        SetBackgroundColor(c.Theme.GetColor("background")).
        SetBorderPadding(2,2,2,2)

    c.layout.AddItem(c.roomWrapper, 0, 1, false).
        AddItem(c.chatView, 0, 4, true)
    c.UI.App.SetFocus(c.msgInput)

}

func (c *ChatScreen) UpdateRoomList(rooms []models.RoomMeta) {
    c.rooms = rooms
    c.roomList.Clear()
    if len(rooms) == 0  {
        if c.noRoomView == nil {
            c.noRoomView = tview.NewTextView().
                SetTextAlign(tview.AlignCenter).
                SetTextColor(c.Theme.GetColor("foreground")).
                SetText("No room available.")
            c.roomPane.AddItem(c.noRoomView, 0, 1, false)
        }
    } else {
        c.roomPane.RemoveItem(c.noRoomView)

    for i, rm := range rooms {
        // The main text shows name, description, lock icon, online count
        line := fmt.Sprintf(
            "%s %s",
            rm.Name,
            formatBoolPasswordProtected(rm.Visibility),
        )
        // store index i in the List for selection
        c.roomList.AddItem(line, "", 0, func(idx int) func() {
            return func() {
                c.selectedRoom = rm
                if c.selectedRoom.Visibility == models.Public {
                    err := c.OnJoinRoom(c.selectedRoom.ID, "")
                    if err != nil {
                        c.UI.ShowError("Join room failed",err.Error(),"OK", 0, nil)
                        return
                    }
                } else {
                    c.joinRoomForm()
                } 
            }
        }(i))
    }
}
}

func (cli *Client) refreshRoomList() {
    roomResp, err := cli.requestRooms(cli.Session.Server.ID)
    cli.UI.App.QueueUpdateDraw(func() {
        if err != nil {
            cli.UI.ShowError("Server Error",err.Error(),"Go back to Browse view", 0, func() {
                cli.UI.Pages.SwitchToPage("browse")
            })
            return
        } else {
            cli.UI.ChatScreen.UpdateRoomList(roomResp.Rooms)
        }
    })  
}


func (c *ChatScreen) showCreateRoomForm() {

    c.modalForm = tview.NewForm()

    bgColor, fieldBg, buttonBg, buttonText, fieldText := c.UI.Theme.FormColors()
    c.modalForm.SetBackgroundColor(bgColor)
    c.modalForm.SetButtonBackgroundColor(buttonBg)
    c.modalForm.SetButtonTextColor(buttonText)
    c.modalForm.SetFieldBackgroundColor(fieldBg)
    c.modalForm.SetFieldTextColor(fieldText)
    c.modalForm.SetLabelColor(c.UI.Theme.GetColor("primary"))
	c.modalForm.SetBorder(true)
    c.modalForm.SetBorderColor(c.UI.Theme.GetColor("border"))
	c.modalForm.SetBorderAttributes(tcell.AttrNone)

    visibilityDropdown := tview.NewDropDown().
        SetLabel("Visibility").
        SetOptions([]string{"Public", "Password Protected", "Private"}, nil)
    
    visibilityDropdown.SetBackgroundColor(c.UI.Theme.GetColor("background"))
    visibilityDropdown.SetFieldBackgroundColor(fieldBg)
    visibilityDropdown.SetFieldTextColor(fieldText)
    visibilityDropdown.SetPrefixTextColor(c.UI.Theme.GetColor("background-light"))
    visibilityDropdown.SetLabelColor(c.UI.Theme.GetColor("primary"))
    visibilityDropdown.SetListStyles(
        tcell.StyleDefault.
		Foreground(fieldText).
		Background(c.UI.Theme.GetColor("background")),
        tcell.StyleDefault.
		Foreground(fieldText).
		Background(c.UI.Theme.GetColor("background-light")),
    )
    visibilityDropdown.SetFocusedStyle(tcell.StyleDefault.
		Foreground(fieldText).
		Background(c.UI.Theme.GetColor("background")))

    c.modalForm.AddInputField("Name", "", 0, nil, nil).
        AddPasswordField("Password (opt)", "", 0, '*', nil).
		AddFormItem(visibilityDropdown).
        AddButton("Save", func() {
            name := c.modalForm.GetFormItemByLabel("Name").(*tview.InputField).GetText()
            pass := c.modalForm.GetFormItemByLabel("Password (opt)").(*tview.InputField).GetText()
			visibilityIndex, _ := c.modalForm.GetFormItemByLabel("Visibility").(*tview.DropDown).GetCurrentOption()
			visibility := models.Visibility(visibilityIndex)
			req := models.CreateRoomRequest{
                ServerID:    c.GetServerId(),
				RoomName:        name,
				Visibility:  visibility,
				PasswordHash: []byte(pass),
			}
			
			sid, err := c.OnCreateRoom(req)
			if err != nil {
				c.UI.ShowError("Create room failed",err.Error(), "OK",0, nil)
				return
			}
            if req.Visibility == models.Private {
                c.UI.ShowToast(fmt.Sprintf("Room created successfully! ID: %s\nThis RoomID will be the only way to access the room. It's been saved under ~/.hillside, encrypted with the room password. DON'T LOSE IT",sid), 0, nil)
                saveEncryptedSID(sid, pass)
            } else {
			    c.UI.ShowToast("Room created successfully! ID: "+sid, 3*time.Second, nil)
            }
            c.UI.Pages.RemovePage("createRoom")
        }).
        AddButton("Cancel", func() {
            c.UI.Pages.RemovePage("createRoom")
        })


    c.modalForm.SetBorder(true).
        SetTitle("[ Create Server ]").
        SetTitleAlign(tview.AlignCenter).
        SetTitleColor(c.UI.Theme.GetColor("primary"))

	mf := func(p tview.Primitive, width, height int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false), width, 1, true).
			AddItem(nil, 0, 1, false)
	}


    c.UI.Pages.AddPage("createRoom", mf(c.modalForm,40,12), true, true)
    c.UI.App.SetFocus(c.modalForm)
}



func (c *ChatScreen) joinRoomForm(){
    c.joinForm = tview.NewForm()

    bgColor, fieldBg, buttonBg, buttonText, fieldText := c.UI.Theme.FormColors()
    c.joinForm.SetBackgroundColor(bgColor)
    c.joinForm.SetButtonBackgroundColor(buttonBg)
    c.joinForm.SetButtonTextColor(buttonText)
    c.joinForm.SetFieldBackgroundColor(fieldBg)
    c.joinForm.SetFieldTextColor(fieldText)
    c.joinForm.SetLabelColor(c.UI.Theme.GetColor("primary"))
	c.joinForm.SetBorder(true)
    c.joinForm.SetBorderColor(c.UI.Theme.GetColor("border"))
	c.joinForm.SetBorderAttributes(tcell.AttrNone)
    c.joinForm.SetButtonsAlign(tview.AlignCenter)

    c.joinForm.AddPasswordField("Password", "", 0, '*', nil).
        AddButton("Join", func() {
            pass := c.joinForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()
			
			
			err := c.OnJoinRoom(c.selectedRoom.ID, pass)
			if err != nil {
				c.UI.ShowError("Join room failed", err.Error(),"OK", 0, nil)
				return
			}
           
            c.UI.Pages.RemovePage("joinRoom")
        }).
        AddButton("Cancel", func() {
            c.UI.Pages.RemovePage("joinRoom")
        })


    c.joinForm.SetBorder(true).
        SetTitle(fmt.Sprintf("[ Join %s ]", c.selectedRoom.Name)).
        SetTitleAlign(tview.AlignCenter).
        SetTitleColor(c.UI.Theme.GetColor("primary"))

	mf := func(p tview.Primitive, width, height int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false), width, 1, true).
			AddItem(nil, 0, 1, false)
	}


    c.UI.Pages.AddPage("joinRoom", mf(c.joinForm,40,8), true, true)
    c.UI.App.SetFocus(c.joinForm)
}