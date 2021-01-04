package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

//Get Youtube data from status
func YtGetStatus(Group, Member int64, Status, Region string) ([]YtDbData, error) {
	var (
		Data  []YtDbData
		list  YtDbData
		rows  *sql.Rows
		limit int
		err   error
		ctx   = context.Background()
		Key   = strconv.Itoa(int(Group*Member)) + Status + Region + "*"
	)
	val := LiveCache.LRange(ctx, Key, 0, -1).Val()
	if len(val) == 0 {
		err = LiveCache.Expire(ctx, Key, 20*time.Minute).Err()
		if err != nil {
			return nil, err
		}

		if (Group != 0 && Status != "live") || (Member != 0 && Status == "past") {
			limit = 3
		} else {
			limit = 2525
		}
		if Region == "" {
			rows, err = DB.Query(`call GetYt(?,?,?,?)`, Member, Group, limit, Status)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
		} else {
			rows, err = DB.Query(`call GetYtByReg(?,?,?)`, Group, Status, Region)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
		}

		for rows.Next() {
			err = rows.Scan(&list.ID, &list.Group, &list.ChannelID, &list.NameEN, &list.NameJP, &list.YoutubeAvatar, &list.VideoID, &list.Title, &list.Thumb, &list.Desc, &list.Schedul, &list.End, &list.Region, &list.Viewers, &list.MemberID, &list.GroupID)
			if err != nil {
				return nil, err
			}
			list.Status = Status
			Data = append(Data, list)
			err = LiveCache.LPush(ctx, Key, list).Err()
			if err != nil {
				return nil, err
			}
		}
	} else {
		for _, result := range val {
			err = json.Unmarshal([]byte(result), &list)
			if err != nil {
				return nil, err
			}
			Data = append(Data, list)
		}
	}

	return Data, nil

}

//Input youtube new video
func (Data *YtDbData) InputYt(MemberID int64) error {
	stmt, err := DB.Prepare(`INSERT INTO Youtube (VideoID,Type,Status,Title,Thumbnails,Description,PublishedAt,ScheduledStart,EndStream,Viewers,Length,VtuberMember_id) values(?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(Data.VideoID, Data.Type, Data.Status, Data.Title, Data.Thumb, Data.Desc, Data.Published, Data.Schedul, Data.End, Data.Viewers, Data.Length, MemberID)
	if err != nil {
		return err
	}

	_, err = res.LastInsertId()
	if err != nil {
		return err
	}

	return nil
}

//Check new video or not
func (Member Member) CheckYtVideo(VideoID string) (*YtDbData, error) {
	var Data YtDbData
	rows, err := DB.Query(`SELECT id,VideoID,Type,Status,Title,Thumbnails,Description,PublishedAt,ScheduledStart,EndStream,Viewers FROM Vtuber.Youtube Where VideoID=? AND VtuberMember_id=?`, VideoID, Member.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Data.ID, &Data.VideoID, &Data.Type, &Data.Status, &Data.Title, &Data.Thumb, &Data.Desc, &Data.Published, &Data.Schedul, &Data.End, &Data.Viewers)
		if err != nil {
			return nil, err
		}
	}
	if Data.ID == 0 {
		return nil, errors.New("VideoID not found in database")
	} else {
		return &Data, nil
	}
}

//Update youtube data
func (Data *YtDbData) UpdateYt(Status string) {
	_, err := DB.Exec(`Update Youtube set Type=?,Status=?,Title=?,Thumbnails=?,Description=?,PublishedAt=?,ScheduledStart=?,EndStream=?,Viewers=?,Length=? where id=?`, Data.Type, Status, Data.Title, Data.Thumb, Data.Desc, Data.Published, Data.Schedul, Data.End, Data.Viewers, Data.Length, Data.ID)
	if err != nil {
		log.Error(err)
	}
}
