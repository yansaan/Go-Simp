package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	config "github.com/JustHumanz/Go-Simp/pkg/config"
	"github.com/go-redis/redis/v8"
	twitterscraper "github.com/n0madic/twitter-scraper"
	log "github.com/sirupsen/logrus"
)

//Public variable
var (
	DB           *sql.DB
	FanartCache  *redis.Client
	LiveCache    *redis.Client
	GeneralCache *redis.Client
)

//Start Database session
func Start(configfile config.ConfigFile) {
	DB = configfile.CheckSQL()
	RedisHost := configfile.Cached.Host + ":" + configfile.Cached.Port
	FanartCache = redis.NewClient(&redis.Options{
		Addr:     RedisHost,
		Password: "",
		DB:       0,
	})

	LiveCache = redis.NewClient(&redis.Options{
		Addr:     RedisHost,
		Password: "",
		DB:       1,
	})

	GeneralCache = redis.NewClient(&redis.Options{
		Addr:     RedisHost,
		Password: "",
		DB:       2,
	})
	log.Info("Database module ready")
}

func GetModule() []string {
	return GeneralCache.LRange(context.Background(), "ModuleInfo", 0, -1).Val()
}

//GetGroup Get all vtuber groupData
func GetGroups() []Group {
	rows, err := DB.Query(`SELECT id,VtuberGroupName,VtuberGroupIcon FROM VtuberGroup`)
	if err != nil {
		log.Error(err)
	}
	defer rows.Close()

	var Data []Group
	for rows.Next() {
		var list Group
		err = rows.Scan(&list.ID, &list.GroupName, &list.IconURL)
		if err != nil {
			log.Error(err)
		}
		Data = append(Data, list)

	}
	return Data
}

//GetMember Get data of Vtuber member
func GetMembers(GroupID int64) []Member {
	var (
		list Member
		Data []Member
	)
	rows, err := DB.Query(`call GetVtuberName(?)`, GroupID)
	if err != nil {
		log.Error(err)
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&list.ID, &list.Name, &list.EnName, &list.JpName, &list.YoutubeID, &list.BiliBiliID, &list.BiliRoomID, &list.Region, &list.TwitterHashtags, &list.BiliBiliHashtags, &list.BiliBiliAvatar, &list.TwitterName, &list.TwitchName, &list.YoutubeAvatar)
		if err != nil {
			log.Error(err)
		}
		if err != nil {
			log.Error(err)
		}
		Data = append(Data, list)
	}
	return Data
}

//gacha is gacha
func gacha() bool {
	return rand.Float32() < 0.5
}

//GetSubsCount Get subs,follow,view,like data from Subscriber
func (Member Member) GetSubsCount() (*MemberSubs, error) {
	var (
		Data MemberSubs
		Key  = strconv.Itoa(int(Member.ID)) + Member.Name
	)

	val, err := GeneralCache.Get(context.Background(), Key).Result()
	if len(val) == 0 || err == redis.Nil {
		rows, err := DB.Query(`SELECT * FROM Subscriber WHERE VtuberMember_id=?`, Member.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			err = rows.Scan(&Data.ID, &Data.YtSubs, &Data.YtVideos, &Data.YtViews, &Data.BiliFollow, &Data.BiliVideos, &Data.BiliViews, &Data.TwFollow, &Data.MemberID)
			if err != nil {
				return nil, err
			}
		}
		err = GeneralCache.Set(context.Background(), Key, Data, 16*time.Minute).Err()
		if err != nil {
			return nil, err
		}

	} else {
		err := json.Unmarshal([]byte(val), &Data)
		if err != nil {
			return nil, err
		}
	}

	return &Data, nil
}

//UpBiliFollow update bilibili state
func (Member *MemberSubs) UpBiliFollow(new int) *MemberSubs {
	Member.BiliFollow = new
	return Member
}

//UpBiliVideo Add bilibili Videos
func (Member *MemberSubs) UpBiliVideo(new int) *MemberSubs {
	Member.BiliVideos = new
	return Member
}

//UpBiliViews Add views
func (Member *MemberSubs) UpBiliViews(new int) *MemberSubs {
	Member.BiliViews = new
	return Member
}

//UpYtSubs update youtube state
func (Member *MemberSubs) UpYtSubs(new int) *MemberSubs {
	Member.YtSubs = new
	return Member
}

//UpYtVideo Update youtube videos
func (Member *MemberSubs) UpYtVideo(new int) *MemberSubs {
	Member.YtVideos = new
	return Member
}

//UpYtViews Update youtube views
func (Member *MemberSubs) UpYtViews(new int) *MemberSubs {
	Member.YtViews = new
	return Member
}

//UptwFollow Update twitter state
func (Member *MemberSubs) UptwFollow(new int) *MemberSubs {
	Member.TwFollow = new
	return Member
}

//UpdateSubs Update Subscriber data
func (Member *MemberSubs) UpdateSubs(State string) {
	if State == "yt" {
		_, err := DB.Exec(`Update Subscriber set Youtube_Subscriber=?,Youtube_Videos=?,Youtube_Views=? Where id=? `, Member.YtSubs, Member.YtVideos, Member.YtViews, Member.ID)
		if err != nil {
			log.Error(err)
		}
	} else if State == "bili" {
		_, err := DB.Exec(`Update Subscriber set BiliBili_Followers=?,BiliBili_Videos=?,BiliBili_Views=? Where id=? `, Member.BiliFollow, Member.BiliVideos, Member.BiliViews, Member.ID)
		if err != nil {
			log.Error(err)
		}
	} else {
		_, err := DB.Exec(`Update Subscriber set Twitter_Followers=? Where id=? `, Member.TwFollow, Member.ID)
		if err != nil {
			log.Error(err)
		}
	}
}

//GetFanart Get Member fanart URL from TBiliBili and Twitter
func GetFanart(GroupID, MemberID int64) DataFanart {
	var (
		Data     DataFanart
		PhotoTmp sql.NullString
		Video    sql.NullString
		rows     *sql.Rows
		err      error
	)

	Twitter := func(State string) {
		rows, err = DB.Query(`Call GetArt(?,?,'twitter')`, GroupID, MemberID)
		if err != nil {
			log.Error(err)
		}

		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&Data.ID, &Data.EnName, &Data.JpName, &Data.PermanentURL, &Data.Author, &PhotoTmp, &Video, &Data.Text)
			if err != nil {
				log.Error(err)
			}
		}
		Data.State = State
	}
	Tbilibili := func(State string) {
		rows, err = DB.Query(`Call GetArt(?,?,'westtaiwan')`, GroupID, MemberID)
		if err != nil {
			log.Error(err)
		}

		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&Data.ID, &Data.EnName, &Data.JpName, &Data.PermanentURL, &Data.Author, &PhotoTmp, &Video, &Data.Text)
			if err != nil {
				log.Error(err)
			}
		}
		Data.State = State
	}

	if gacha() {
		Twitter("Twitter")
	} else {
		Tbilibili("TBiliBili")
		if Data.ID == 0 {
			log.Warn("Tbilibili nill")
			Twitter("Twitter")
		}
	}

	Data.Videos = Video.String
	Data.Photos = strings.Fields(PhotoTmp.String)
	return Data

}

func (Data DataFanart) DeleteFanart() error {
	if Data.State == "Twitter" {
		stmt, err := DB.Prepare(`DELETE From Twitter WHERE id=?`)
		if err != nil {
			log.Error(err)
		}
		defer stmt.Close()

		stmt.Exec(Data.ID)
		return nil
	} else {
		stmt, err := DB.Prepare(`DELETE From TBiliBili WHERE id=?`)
		if err != nil {
			log.Error(err)
		}
		defer stmt.Close()

		stmt.Exec(Data.ID)
		return nil
	}
}

func (Member Member) CheckMemberFanart(Data *twitterscraper.Result) (bool, error) {
	var (
		id     int
		videos string
	)
	err := DB.QueryRow(`SELECT id FROM Twitter WHERE PermanentURL=?`, Data.PermanentURL).Scan(&id)
	if err == sql.ErrNoRows {
		log.WithFields(log.Fields{
			"Name":    Member.EnName,
			"Hashtag": Member.TwitterHashtags,
		}).Info("New Fanart")

		stmt, err := DB.Prepare(`INSERT INTO Twitter (PermanentURL,Author,Likes,Photos,Videos,Text,TweetID,VtuberMember_id) values(?,?,?,?,?,?,?,?)`)
		if err != nil {
			return false, err
		}
		defer stmt.Close()

		if len(Data.Videos) > 0 {
			videos = Data.Videos[0].URL
		}

		res, err := stmt.Exec(Data.PermanentURL, Data.Username, Data.Likes, strings.Join(Data.Photos, "\n"), videos, Data.Text, Data.ID, Member.ID)
		if err != nil {
			return false, err
		}

		_, err = res.LastInsertId()
		if err != nil {
			return false, err
		}
		return true, nil
	} else {
		if !config.GoSimpConf.LowResources {
			//update like
			log.WithFields(log.Fields{
				"Name":    Member.EnName,
				"Hashtag": Member.TwitterHashtags,
				"Likes":   Data.Likes,
			}).Info("Update like")
			_, err := DB.Exec(`Update Twitter set Likes=? Where id=? `, Data.Likes, id)
			if err != nil {
				return false, err
			}
		}
	}
	return false, nil
}

//GetChannelID Get Channel id from Discord ChannelID and VtuberGroupID
func GetChannelID(DiscordChannelID string, GroupID int64) int {
	var ChannelID int
	err := DB.QueryRow("SELECT id from Channel where DiscordChannelID=? AND VtuberGroup_id=?", DiscordChannelID, GroupID).Scan(&ChannelID)
	if err != nil {
		log.Error(err)
	}
	return ChannelID
}

//Adduser form `tag me command`
func (Data *UserStruct) Adduser() error {
	ChannelID := GetChannelID(Data.Channel_ID, Data.Group.ID)
	tmp := CheckUser(Data.DiscordID, Data.Member.ID, ChannelID)
	if tmp {
		return errors.New("Already registered")
	} else {
		stmt, err := DB.Prepare(`INSERT INTO User (DiscordID,DiscordUserName,Human,Reminder,VtuberMember_id,Channel_id) values(?,?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		res, err := stmt.Exec(Data.DiscordID, Data.DiscordUserName, Data.Human, Data.Reminder, Data.Member.ID, ChannelID)
		if err != nil {
			return err
		}

		_, err = res.LastInsertId()
		if err != nil {
			return err
		}

		return nil
	}
}

func (Data *UserStruct) SendToCache(MessageID string) error {
	err := GeneralCache.Set(context.Background(), MessageID, Data, -1).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetChannelMessage(MessageID string) *UserStruct {
	var (
		data UserStruct
	)
	val := GeneralCache.Get(context.Background(), MessageID).Val()
	err := json.Unmarshal([]byte(val), &data)
	if err != nil {
		return nil
	}
	return &data
}

//UpdateReminder Update reminder time
func (Data UserStruct) UpdateReminder() error {
	ChannelID := GetChannelID(Data.Channel_ID, Data.Group.ID)
	tmp := CheckUser(Data.DiscordID, Data.Member.ID, ChannelID)
	if tmp {
		_, err := DB.Exec(`Update User set Reminder=? where DiscordID=? And VtuberMember_id=? And Channel_id=?`, Data.Reminder, Data.DiscordID, Data.Member.ID, ChannelID)
		if err != nil {
			return err
		}
	} else {
		return errors.New("User not tag " + strconv.Itoa(int(Data.Member.ID)))
	}
	return nil
}

//Deluser Delete user from `del` command
func (Data UserStruct) Deluser() error {
	ChannelID := GetChannelID(Data.Channel_ID, Data.Group.ID)
	tmp := CheckUser(Data.DiscordID, Data.Member.ID, ChannelID)
	if tmp {
		stmt, err := DB.Prepare(`DELETE From User WHERE DiscordID=? AND VtuberMember_id=? AND Channel_id=?`)
		if err != nil {
			log.Error(err)
		}
		defer stmt.Close()

		stmt.Exec(Data.DiscordID, Data.Member.ID, ChannelID)
		return nil
	} else {
		return errors.New("Already removed or you not in tag list")
	}
}

//CheckUser Check user if already tagged
func CheckUser(DiscordID string, MemberID int64, ChannelChannelID int) bool {
	var tmp int
	row := DB.QueryRow("SELECT id FROM User WHERE DiscordID=? AND VtuberMember_id=? AND Channel_id=?", DiscordID, MemberID, ChannelChannelID)
	err := row.Scan(&tmp)
	if err != nil || err == sql.ErrNoRows {
		return false
	} else {
		return true
	}
}

//Add new discord channel from `enable` command
func (Data *DiscordChannel) AddChannel() error {
	if Data.Dynamic {
		Data.SetNewUpcoming(false).SetLiveOnly(true)
	}
	stmt, err := DB.Prepare(`INSERT INTO Channel (DiscordChannelID,Type,LiveOnly,NewUpcoming,Dynamic,VtuberGroup_id) values(?,?,?,?,?,?)`)
	if err != nil {
		return err
	}

	defer stmt.Close()
	res, err := stmt.Exec(Data.ChannelID, Data.TypeTag, Data.LiveOnly, Data.NewUpcoming, Data.Dynamic, Data.Group.ID)
	if err != nil {
		return err
	}

	_, err = res.LastInsertId()
	if err != nil {
		return err
	}

	if Data.Dynamic {
		return errors.New("force to set Dynamic")
	}

	return nil
}

//delete discord channel from `disable` command
func (Data *DiscordChannel) DelChannel(errmsg string) error {
	match, err := regexp.MatchString("Unknown|403|Delete|Missing", errmsg)
	if err != nil {
		return err
	}

	if match {
		log.Info("Delete Discord Channel ", Data.ChannelID)
		var (
			ID int64
		)

		row := DB.QueryRow("SELECT id FROM Channel WHERE DiscordChannelID=? AND VtuberGroup_id=?", Data.ChannelID, Data.Group.ID)
		err := row.Scan(&ID)
		if err != nil {
			return err
		}

		rows, err := DB.Query(`SELECT id FROM User Where Channel_id=?`, ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var tmp int64
			err = rows.Scan(&tmp)
			if err != nil {
				return err
			}

			stmt, err := DB.Prepare(`DELETE From User WHERE id=?`)
			if err != nil {
				return err
			}
			defer stmt.Close()
			stmt.Exec(tmp)
		}

		stmt, err := DB.Prepare(`DELETE From Channel WHERE DiscordChannelID=? AND VtuberGroup_id=?`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		stmt.Exec(Data.ChannelID, Data.Group.ID)
	}
	return nil
}

//update discord channel type from `update` command
func (Data *DiscordChannel) UpdateChannel(UpdateType string) error {
	var (
		channeltype int
		liveonly    bool
		newupcoming bool
		dynamic     bool
	)
	rows, err := DB.Query(`SELECT Type,LiveOnly,NewUpcoming,Dynamic FROM Channel WHERE VtuberGroup_id=? AND DiscordChannelID=?`, Data.Group.ID, Data.ChannelID)
	if err != nil {
		log.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&channeltype, &liveonly, &newupcoming, &dynamic)
		if err != nil {
			log.Error(err)
		}
	}
	if channeltype == 1 {
		liveonly = false
		newupcoming = false
		dynamic = false
	}

	if UpdateType == "Type" {
		if channeltype == Data.TypeTag {
			return errors.New("Already set that type on this channel")
		} else {
			_, err := DB.Exec(`Update Channel set Type=?,LiveOnly=?,NewUpcoming=?,Dynamic=? Where VtuberGroup_id=? AND DiscordChannelID=?`, Data.TypeTag, liveonly, newupcoming, dynamic, Data.Group.ID, Data.ChannelID)
			if err != nil {
				return err
			}
		}
	} else if UpdateType == "LiveOnly" {
		if liveonly == Data.LiveOnly {
			return errors.New("Already set LiveOnly on this channel")
		} else {
			_, err := DB.Exec(`Update Channel set Type=?,LiveOnly=?,NewUpcoming=?,Dynamic=? Where VtuberGroup_id=? AND DiscordChannelID=?`, channeltype, Data.LiveOnly, newupcoming, dynamic, Data.Group.ID, Data.ChannelID)
			if err != nil {
				return err
			}
		}
	} else if UpdateType == "Dynamic" {
		if dynamic == Data.Dynamic {
			return errors.New("Already set Dynamic on this channel")
		} else if newupcoming && dynamic {
			newupcoming = false
			liveonly = true
		} else {
			_, err := DB.Exec(`Update Channel set Type=?,LiveOnly=?,NewUpcoming=?,Dynamic=? Where VtuberGroup_id=? AND DiscordChannelID=?`, channeltype, liveonly, newupcoming, Data.Dynamic, Data.Group.ID, Data.ChannelID)
			if err != nil {
				return err
			}
		}
	} else {
		if newupcoming == Data.NewUpcoming {
			return errors.New("Already set NewUpcoming on this channel")
		} else {
			_, err := DB.Exec(`Update Channel set Type=?,LiveOnly=?,NewUpcoming=?,Dynamic=? Where VtuberGroup_id=? AND DiscordChannelID=?`, channeltype, liveonly, Data.NewUpcoming, dynamic, Data.Group.ID, Data.ChannelID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//Get DiscordChannelID from VtuberGroup
func (Data Group) GetChannelByGroup() []DiscordChannel {
	var (
		list        string
		id          int64
		ChannelData []DiscordChannel
	)
	rows, err := DB.Query(`SELECT id,DiscordChannelID FROM Channel WHERE VtuberGroup_id=? group by DiscordChannelID`, Data.ID)
	if err != nil {
		log.Error(err)
	}

	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&id, &list)
		if err != nil {
			log.Error(err)
		}
		ChannelData = append(ChannelData, DiscordChannel{
			ID:        id,
			ChannelID: list,
		})
	}
	return ChannelData
}

//Check Discord Channel from VtuberGroup
func (Data *DiscordChannel) ChannelCheck() bool {
	var tmp int
	row := DB.QueryRow("SELECT id FROM Channel WHERE VtuberGroup_id=? AND DiscordChannelID=?", Data.Group.ID, Data.ChannelID)
	err := row.Scan(&tmp)
	if err != nil || err == sql.ErrNoRows {
		return false
	} else {
		return true
	}
}

//Check enable or disable discord channel from `tag,del` command
func CheckChannelEnable(ChannelID, VtuberName string, GroupID int64) bool {
	var DiscordChannelID string
	row := DB.QueryRow("Select DiscordChannelID FROM Channel Inner join VtuberGroup on VtuberGroup.id = Channel.VtuberGroup_id inner Join VtuberMember on VtuberMember.VtuberGroup_id = VtuberGroup.id Where Channel.DiscordChannelID=? AND (VtuberMember.VtuberName_EN=? OR VtuberMember.VtuberName_JP=? OR VtuberGroup.id=?) group by Channel.id", ChannelID, VtuberName, VtuberName, GroupID)
	err := row.Scan(&DiscordChannelID)
	if err != nil || err == sql.ErrNoRows {
		return false
	} else {
		return true
	}
}

//Get userinfo(tags) from discord channel
func UserStatus(UserID, Channel string) [][]string {
	var (
		GroupName  string
		VtuberName string
		Reminder   string
		taglist    [][]string
	)
	rows, err := DB.Query(`SELECT VtuberGroupName,VtuberName,User.Reminder FROM User INNER JOIN VtuberMember ON User.VtuberMember_id=VtuberMember.id Join VtuberGroup ON VtuberGroup.id = VtuberMember.VtuberGroup_id Inner Join Channel on Channel.id=User.Channel_id WHERE DiscordChannelID=? And DiscordID=?`, Channel, UserID)
	if err != nil {
		log.Error(err)
	}

	defer rows.Close()
	for rows.Next() {
		tmpReminder := 0
		err = rows.Scan(&GroupName, &VtuberName, &tmpReminder)
		if err != nil {
			log.Error(err)
		}

		if tmpReminder == 0 {
			Reminder = "None"
		} else {
			Reminder = strconv.Itoa(tmpReminder) + " Minutes"
		}

		taglist = append(taglist, []string{GroupName, VtuberName, Reminder})
	}
	return taglist
}

//Get Discord channel status
func ChannelStatus(ChannelID string) []DiscordChannel {
	var (
		Data []DiscordChannel
	)
	rows, err := DB.Query(`SELECT Channel.id,VtuberGroupName,Channel.Type,Channel.LiveOnly,Channel.NewUpcoming,Channel.Dynamic FROM Channel INNER JOIn VtuberGroup on VtuberGroup.id=Channel.VtuberGroup_id WHERE DiscordChannelID=?`, ChannelID)
	if err != nil {
		log.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id   int64
			tmp  string
			tmp2 int
			tmp3 bool
			tmp4 bool
			tmp5 bool
		)
		err = rows.Scan(&id, &tmp, &tmp2, &tmp3, &tmp4, &tmp5)
		if err != nil {
			log.Error(err)
		}
		/*
			if tmp3 {
				LiveOnly = append(LiveOnly, "Enabled")
			} else {
				LiveOnly = append(LiveOnly, "Disabled")
			}
			if tmp4 {
				NewUpcoming = append(NewUpcoming, "Enabled")
			} else {
				NewUpcoming = append(NewUpcoming, "Disabled")
			}
		*/
		Data = append(Data, DiscordChannel{
			ID: id,
			Group: Group{
				GroupName: tmp,
			},
			TypeTag:     tmp2,
			LiveOnly:    tmp3,
			NewUpcoming: tmp4,
			Dynamic:     tmp5,
		})
	}
	return Data
}

//ChannelTag get channel tags from `channel tags` command
func ChannelTag(MemberID int64, typetag int, Options string) []DiscordChannel {
	var (
		Data    []DiscordChannel
		tmp     int64
		tmp2    string
		tmp3    bool
		GroupID int64
		rows    *sql.Rows
		err     error
	)
	if Options == "NotLiveOnly" {
		rows, err = DB.Query(`Select Channel.id,DiscordChannelID,Dynamic,VtuberGroup.id FROM Channel Inner join VtuberGroup on VtuberGroup.id = Channel.VtuberGroup_id inner Join VtuberMember on VtuberMember.VtuberGroup_id = VtuberGroup.id Where VtuberMember.id=? AND (Channel.type=2 OR Channel.type=3) AND LiveOnly=0`, MemberID)
		if err != nil {
			log.Error(err)
		}
		defer rows.Close()

	} else if Options == "NewUpcoming" {
		rows, err = DB.Query(`Select Channel.id,DiscordChannelID,Dynamic,VtuberGroup.id FROM Channel Inner join VtuberGroup on VtuberGroup.id = Channel.VtuberGroup_id inner Join VtuberMember on VtuberMember.VtuberGroup_id = VtuberGroup.id Where VtuberMember.id=? AND (Channel.type=2 OR Channel.type=3) AND NewUpcoming=1`, MemberID)
		if err != nil {
			log.Error(err)
		}
		defer rows.Close()
	} else {
		rows, err = DB.Query(`Select Channel.id,DiscordChannelID,Dynamic,VtuberGroup.id FROM Channel Inner join VtuberGroup on VtuberGroup.id = Channel.VtuberGroup_id inner Join VtuberMember on VtuberMember.VtuberGroup_id = VtuberGroup.id Where VtuberMember.id=? AND (Channel.type=? OR Channel.type=3)`, MemberID, typetag)
		if err != nil {
			log.Error(err)
		}
		defer rows.Close()
	}
	for rows.Next() {
		err = rows.Scan(&tmp, &tmp2, &tmp3, &GroupID)
		if err != nil {
			log.Error(err)
		}
		Data = append(Data, DiscordChannel{
			ID:        tmp,
			ChannelID: tmp2,
			Dynamic:   tmp3,
			Member: Member{
				ID: MemberID,
			},
			Group: Group{
				ID: GroupID,
			},
		})
	}
	return Data
}

func (Data *DiscordChannel) PushReddis() {
	err := GeneralCache.LPush(context.Background(), Data.VideoID, Data).Err()
	if err != nil {
		log.Error(err)
	}
}

func GetLiveNotifMsg(Key string) []DiscordChannel {
	var (
		Data []DiscordChannel
	)
	val := GeneralCache.LRange(context.Background(), Key, 0, -1).Val()
	for _, v := range val {
		var ChannelData DiscordChannel
		err := json.Unmarshal([]byte(v), &ChannelData)
		if err != nil {
			log.Error(err)
		}
		Data = append(Data, ChannelData)
	}
	rederr := GeneralCache.Del(context.Background(), Key).Err()
	if rederr != nil {
		log.Error(rederr)
	}

	return Data
}

//GetUserList GetUser tags
func (Data *DiscordChannel) GetUserList(ctx context.Context) ([]string, error) {
	var (
		DataUser      []string
		DiscordUserID string
		Type          bool
		Key           = Data.Member.Name + strconv.Itoa(int(Data.ID))
	)
	val2, err := LiveCache.Get(ctx, Key).Result()
	if err == redis.Nil {
		rows, err := DB.Query(`SELECT DiscordID,Human From User WHERE Channel_id=? And VtuberMember_id=?`, Data.ID, Data.Member.ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			err = rows.Scan(&DiscordUserID, &Type)
			if err != nil {
				return nil, err
			}
			if Type {
				DataUser = append(DataUser, "<@"+DiscordUserID+">")
			} else {
				DataUser = append(DataUser, "<@&"+DiscordUserID+">")
			}
		}
		if len(DataUser) > 0 {
			err = LiveCache.Set(ctx, Key, strings.Join(DataUser, ","), 17*time.Minute).Err()
			if err != nil {
				return nil, err
			}
		}
	} else if err != nil {
		return nil, err
	} else {
		if val2 == "" {
			DataUser = nil
		} else {
			DataUser = strings.Split(val2, ",")
		}
	}
	return DataUser, nil
}

//get Reminder tags
func GetUserReminderList(ChannelIDDiscord int64, Member int64, Reminder int) []string {
	var (
		UserTagsList  []string
		DiscordUserID string
		Type          bool
	)
	rows, err := DB.Query(`SELECT DiscordID,Human From User WHERE Channel_id=? And VtuberMember_id=? And Reminder=?`, ChannelIDDiscord, Member, Reminder)
	if err != nil {
		log.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&DiscordUserID, &Type)
		if err != nil {
			log.Error(err)
		}
		if Type {
			UserTagsList = append(UserTagsList, "<@"+DiscordUserID+">")
		} else {
			UserTagsList = append(UserTagsList, "<@&"+DiscordUserID+">")
		}
	}
	return UserTagsList
}

//Scrapping twitter followers
func (Data Member) GetTwitterFollow() (twitterscraper.Profile, error) {
	twitterscraper.SetProxy(config.GoSimpConf.MultiTOR)
	profile, err := twitterscraper.GetProfile(Data.TwitterName)
	if err != nil {
		return twitterscraper.Profile{}, err
	}
	return profile, nil
}

//GetRanChannel get random id channel
func GetRanChannel() string {
	var tmp string
	row := DB.QueryRow("SELECT DiscordChannelID FROM Vtuber.Channel ORDER BY RAND() LIMIT 1")
	err := row.Scan(&tmp)
	if err != nil {
		log.Error(err)
	}
	return tmp
}
