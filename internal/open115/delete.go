package open115

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/xhofe/115-sdk-go"
)

type remoteEntryRef struct {
	ID       string
	ParentID string
	Name     string
	IsDir    bool
}

func (u *Uploader) findEntryByName(ctx context.Context, parentID, name string) (*remoteEntryRef, error) {
	items, err := u.listDirItems(ctx, parentID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Fn == name {
			return &remoteEntryRef{ID: item.Fid, ParentID: item.Pid, Name: item.Fn, IsDir: item.Fc == "0"}, nil
		}
	}
	return nil, nil
}

func (u *Uploader) ResolveEntry(ctx context.Context, remotePath string) (*remoteEntryRef, error) {
	if u == nil || u.service == nil {
		return nil, fmt.Errorf("open115 uploader not initialized")
	}
	cleaned := normalizeUploadPath(remotePath)
	if cleaned == "/" {
		return &remoteEntryRef{ID: u.rootID(), ParentID: "", Name: "/", IsDir: true}, nil
	}
	currentID := u.rootID()
	trimmed := strings.Trim(strings.TrimPrefix(cleaned, "/"), "/")
	segments := strings.Split(trimmed, "/")
	var current *remoteEntryRef
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		entry, err := u.findEntryByName(ctx, currentID, seg)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			return nil, fmt.Errorf("远端条目不存在: %s", cleaned)
		}
		current = entry
		currentID = entry.ID
	}
	if current == nil {
		return nil, fmt.Errorf("远端条目不存在: %s", cleaned)
	}
	return current, nil
}

func (u *Uploader) DeleteRemote(ctx context.Context, remotePath string) error {
	if u == nil || u.service == nil {
		return fmt.Errorf("open115 uploader not initialized")
	}
	entry, err := u.ResolveEntry(ctx, remotePath)
	if err != nil {
		return err
	}
	client, err := u.service.Client()
	if err != nil {
		return err
	}
	return CallNoReturn(ctx, u.Pacer, "DelFile", defaultMaxRetries, func() error {
		_, err := client.DelFile(ctx, &sdk.DelFileReq{FileIDs: entry.ID, ParentID: entry.ParentID})
		return err
	})
}

