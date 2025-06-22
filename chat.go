package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    
    "strings"
    "time"

    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
)

type InitialMessage struct {
    Sender     string `json:"sender"`
    AliceIK    string `json:"alice_ik"`
    AliceEKa   string `json:"alice_eka"`
    Nonce      string `json:"nonce"`
    Ciphertext string `json:"ciphertext"`
}

type ServerMessage struct {
    Message      InitialMessage `json:"message"`
    MessagesLeft int            `json:"messages_left"`
}

func main() {
    app := tview.NewApplication()
    chatView := tview.NewTextView().
        SetDynamicColors(true).
        SetChangedFunc(func() { app.Draw() })

    var username, recipient string
    var errorText *tview.TextView
    done := make(chan struct{})

    errorText = tview.NewTextView().SetText("").SetTextColor(tcell.ColorRed)

    form := tview.NewForm().
        AddInputField("Your name", "", 20, nil, func(text string) { username = strings.TrimSpace(strings.ToLower(text)) }).
        AddInputField("Recipient", "", 20, nil, func(text string) { recipient = strings.TrimSpace(strings.ToLower(text)) }).
        AddButton("Start Chat", func() {
            if username == recipient {
                errorText.SetText("Sender and recipient must be different!")
                return
            }
            if !((username == "alice" && recipient == "bob") || (username == "bob" && recipient == "alice")) {
                errorText.SetText("Only alice <-> bob chats are allowed!")
                return
            }
            errorText.SetText("")
            close(done)
        })
    form.SetBorder(true).SetTitle("Login").SetTitleAlign(tview.AlignLeft)

    flexForm := tview.NewFlex().SetDirection(tview.FlexRow).
        AddItem(form, 0, 1, true).
        AddItem(errorText, 1, 1, false)

    go func() {
        <-done
        app.QueueUpdateDraw(func() {
            app.SetRoot(buildChatUI(app, chatView, username, recipient), true)
        })
    }()

    if err := app.SetRoot(flexForm, true).Run(); err != nil {
        panic(err)
    }
}

func buildChatUI(app *tview.Application, chatView *tview.TextView, username, recipient string) tview.Primitive {
    var input *tview.InputField
    input = tview.NewInputField().
        SetLabel("Type a message: ").
        SetDoneFunc(func(key tcell.Key) {
            if key == tcell.KeyEnter {
                msg := input.GetText()
                if msg != "" {
                    sendMessage(username, recipient, msg)
                    chatView.Write([]byte(fmt.Sprintf("%s: %s\n", username, msg)))
                    input.SetText("")
                }
            }
        })

    go func() {
        for {
            time.Sleep(2 * time.Second)
            sender, msg, err := checkMessages(username)
            if err == nil && msg != "" {
                app.QueueUpdateDraw(func() {
                    chatView.Write([]byte(fmt.Sprintf("%s: %s\n", sender, msg)))
                })
            }
        }
    }()

    flex := tview.NewFlex().SetDirection(tview.FlexRow).
        AddItem(tview.NewTextView().SetText(fmt.Sprintf("Chat with %s", recipient)), 1, 1, false).
        AddItem(chatView, 0, 1, false).
        AddItem(input, 1, 1, true)
    return flex
}

func sendMessage(sender, recipient, msg string) {
    m := InitialMessage{
        Sender:     sender,
        AliceIK:    "",
        AliceEKa:   "",
        Nonce:      "",
        Ciphertext: msg,
    }
    data, _ := json.Marshal(m)
    http.Post(fmt.Sprintf("http://localhost:8080/send/%s", recipient), "application/json", bytes.NewBuffer(data))
}

func checkMessages(username string) (string, string, error) {
    resp, err := http.Get(fmt.Sprintf("http://localhost:8080/messages/%s", username))
    if err != nil {
        return "", "", err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return "", "", io.EOF
    }
    var sm ServerMessage
    if err := json.NewDecoder(resp.Body).Decode(&sm); err != nil {
        return "", "", err
    }
    return sm.Message.Sender, sm.Message.Ciphertext, nil
} 