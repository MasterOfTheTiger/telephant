package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/muesli/telephant/accounts"
	"github.com/therecipe/qt/core"
)

// Message holds the data for a message
type Message struct {
	core.QObject

	Name          string
	MessageID     string
	PostURL       string
	Author        string
	AuthorURL     string
	AuthorID      string
	Avatar        string
	Body          string
	CreatedAt     time.Time
	Actor         string
	ActorName     string
	ActorID       string
	Reply         bool
	ReplyToID     string
	ReplyToAuthor string
	Forward       bool
	Mention       bool
	Like          bool
	Followed      bool
	Following     bool
	FollowedBy    bool
	MediaPreview  []string
	MediaURL      []string
	Editing       bool
	Liked         bool
	Shared        bool
}

var (
	store           = make(map[string]*Message)
	modelReferences = make(map[string][]*MessageModel)
	mutex           sync.RWMutex
)

func addMessage(model *MessageModel, m *Message) {
	fmt.Println("store", m.MessageID)

	mutex.Lock()
	defer mutex.Unlock()

	store[m.MessageID] = m
	modelReferences[m.MessageID] = append(modelReferences[m.MessageID], model)
}

func removeMessage(model *MessageModel, m *Message) {
	fmt.Println("remove", m.MessageID)

	mutex.RLock()
	ref := modelReferences[m.MessageID]
	mutex.RUnlock()

	var models []*MessageModel
	for _, v := range ref {
		if v == model {
			continue
		}

		models = append(models, v)
	}

	mutex.Lock()
	defer mutex.Unlock()

	if len(models) == 0 {
		// last reference to message has been deleted
		delete(modelReferences, m.MessageID)
		delete(store, m.MessageID)
	} else {
		modelReferences[m.MessageID] = models
	}
}

func deleteMessage(id string) {
	fmt.Println("delete", id)

	mutex.RLock()
	ref := modelReferences[id]
	mutex.RUnlock()

	for _, v := range ref {
		// v.RemoveMessageID(id)
		for idx, m := range v.Messages() {
			if m.MessageID == id {
				trow := len(v.Messages()) - 1 - idx
				fmt.Println("Found message, deleting from model...", idx, trow)
				v.RemoveMessage(trow)
				break
			}
		}
	}

	fmt.Println("done deleting")
}

func getMessage(id string) *Message {
	mutex.RLock()
	defer mutex.RUnlock()

	return store[id]
}

// messageFromEvent creates a new Message object from an incoming MessageEvent.
func messageFromEvent(event accounts.MessageEvent) *Message {
	p := getMessage(event.Post.MessageID)
	if p == nil {
		p = NewMessage(nil)
	}

	p.Forward = event.Forward
	p.Mention = event.Mention
	p.Like = event.Like
	p.Followed = event.Followed
	p.Reply = event.Reply

	if event.Post.MessageID != "" {
		p.MessageID = event.Post.MessageID
		p.PostURL = event.Post.URL
		p.Name = event.Post.AuthorName
		p.Author = event.Post.Author
		p.AuthorURL = event.Post.AuthorURL
		p.AuthorID = event.Post.AuthorID
		p.Avatar = event.Post.Avatar
		p.Body = strings.TrimSpace(event.Post.Body)
		p.CreatedAt = event.Post.CreatedAt
		p.ReplyToID = event.Post.ReplyToID
		p.ReplyToAuthor = event.Post.ReplyToAuthor
		p.Actor = event.Post.Actor
		p.ActorName = event.Post.ActorName
		p.ActorID = event.Post.ActorID
		p.Liked = event.Post.Liked
		p.Shared = event.Post.Shared

		// parse attachments
		p.MediaPreview = []string{}
		p.MediaURL = []string{}
		if len(event.Media) > 0 {
			for _, v := range event.Media {
				p.MediaPreview = append(p.MediaPreview, v.Preview)
				p.MediaURL = append(p.MediaURL, v.URL)
			}
		}
	}

	if event.Followed {
		p.MessageID = event.Follow.Account
		p.Actor = event.Follow.Account
		p.ActorName = event.Follow.Name
		p.Avatar = event.Follow.Avatar
		p.AuthorURL = event.Follow.ProfileURL
		p.AuthorID = event.Follow.ProfileID
		p.Following = event.Follow.Following
		p.FollowedBy = event.Follow.FollowedBy
	}

	if p.MessageID == "" {
		spw := &spew.ConfigState{Indent: "  ", DisableCapacities: true, DisablePointerAddresses: true}
		log.Println("Invalid message received:", spw.Sdump(event))
	}
	return p
}
