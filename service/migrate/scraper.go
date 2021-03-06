package main

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	database "github.com/JustHumanz/Go-Simp/pkg/database"
	network "github.com/JustHumanz/Go-Simp/pkg/network"
	"github.com/JustHumanz/Go-Simp/service/fanart/twitter"
	bilibili "github.com/JustHumanz/Go-Simp/service/livestream/bilibili/live"
	youtube "github.com/JustHumanz/Go-Simp/service/livestream/youtube"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/nicklaw5/helix"
	log "github.com/sirupsen/logrus"
)

func TwitterFanart() {
	scraper := twitterscraper.New()
	scraper.SetProxy(configfile.MultiTOR)
	scraper.SetSearchMode(twitterscraper.SearchLatest)
	for _, Group := range database.GetGroups() {
		_, err := twitter.CreatePayload(Group, scraper, 100)
		if err != nil {
			log.WithFields(log.Fields{
				"Group": Group.GroupName,
			}).Error(err)
		}
	}
}

func FilterYt(Dat database.Member, wg *sync.WaitGroup) {
	VideoID := youtube.GetRSS(Dat.YoutubeID)
	defer wg.Done()
	body, err := network.Curl("https://www.googleapis.com/youtube/v3/videos?part=statistics,snippet,liveStreamingDetails&fields=items(snippet(publishedAt,title,description,thumbnails(standard),channelTitle,liveBroadcastContent),liveStreamingDetails(scheduledStartTime,actualEndTime),statistics(viewCount))&id="+strings.Join(VideoID, ",")+"&key="+YoutubeToken, nil)
	if err != nil {
		log.Error(err, string(body))
	}
	var (
		Data    YtData
		Viewers string
		yttype  string
	)
	err = json.Unmarshal(body, &Data)
	if err != nil {
		log.Error(err)
	}
	for i := 0; i < len(Data.Items); i++ {
		if Cover, _ := regexp.MatchString("(?m)(cover|song|feat|music|mv)", Data.Items[i].Snippet.Title); Cover {
			yttype = "Covering"
		} else if Chat, _ := regexp.MatchString("(?m)(free|chat|room)", Data.Items[i].Snippet.Title); Chat {
			yttype = "ChatRoom"
		} else {
			yttype = "Streaming"
		}

		YtData, err := Dat.CheckYtVideo(VideoID[i])
		if err != nil {
			log.Error(err)
		}

		if YtData != nil {
			continue
		} else {
			log.Info("New video")
			//verify
			if Data.Items[i].LiveDetails.Viewers == "" {
				Viewers = Data.Items[i].Statistics.ViewCount
			} else {
				Viewers = Data.Items[i].LiveDetails.Viewers
			}
			NewData := database.YtDbData{
				Status:    Data.Items[i].Snippet.VideoStatus,
				VideoID:   VideoID[i],
				Title:     Data.Items[i].Snippet.Title,
				Thumb:     "http://i3.ytimg.com/vi/" + VideoID[i] + "/maxresdefault.jpg",
				Desc:      Data.Items[i].Snippet.Description,
				Schedul:   Data.Items[i].LiveDetails.StartTime,
				Published: Data.Items[i].Snippet.PublishedAt,
				End:       Data.Items[i].LiveDetails.EndTime,
				Type:      yttype,
				Viewers:   Viewers,
				MemberID:  Dat.ID,
			}

			if Data.Items[i].Snippet.VideoStatus != "upcoming" || Data.Items[i].Snippet.VideoStatus != "live" {
				NewData.Status = "past"
				NewData.InputYt()
			} else {
				NewData.InputYt()
			}
		}
	}
}

func (Data Member) YtAvatar() string {
	var (
		datasubs Subs
	)
	if Data.YtID != "" {
		body, err := network.Curl("https://www.googleapis.com/youtube/v3/channels?part=snippet&id="+Data.YtID+"&key="+YoutubeToken, nil)
		if err != nil {
			log.Error(err)
		}
		err = json.Unmarshal(body, &datasubs)
		if err != nil {
			log.Error(err)
		}
		return datasubs.Items[0].Snippet.Thumbnails.High.URL
	} else {
		return ""
	}
}

func (Data Member) GetYtSubs() Subs {
	var (
		datasubs Subs
	)
	if Data.YtID != "" {
		body, err := network.Curl("https://www.googleapis.com/youtube/v3/channels?part=statistics&id="+Data.YtID+"&key="+YoutubeToken, nil)
		if err != nil {
			log.Error(err)
		}
		err = json.Unmarshal(body, &datasubs)
		if err != nil {
			log.Error(err)
		}
		return datasubs
	} else {
		return datasubs.Default()
	}
}

func (Data Member) GetBiliFolow() BiliStat {
	var (
		wg   sync.WaitGroup
		stat BiliStat
	)
	if Data.BiliRoomID != 0 {
		wg.Add(3)
		go func() {
			body, curlerr := network.CoolerCurl("https://api.bilibili.com/x/relation/stat?vmid="+strconv.Itoa(Data.BiliBiliID), BiliBiliSession)
			if curlerr != nil {
				log.Error(curlerr)
			}
			err := json.Unmarshal(body, &stat.Follow)
			if err != nil {
				log.Error(err)
			}
			defer wg.Done()
		}()

		go func() {
			body, curlerr := network.CoolerCurl("https://api.bilibili.com/x/space/upstat?mid="+strconv.Itoa(Data.BiliBiliID), BiliBiliSession)
			if curlerr != nil {
				log.Error(curlerr)
			}
			err := json.Unmarshal(body, &stat.Like)
			if err != nil {
				log.Error(err)
			}
			defer wg.Done()
		}()

		go func() {
			baseurl := "https://api.bilibili.com/x/space/arc/search?mid=" + strconv.Itoa(Data.BiliBiliID) + "&ps=100"
			url := []string{baseurl + "&tid=1", baseurl + "&tid=3", baseurl + "&tid=4"}
			for f := 0; f < len(url); f++ {
				body, curlerr := network.CoolerCurl(url[f], BiliBiliSession)
				if curlerr != nil {
					log.Error(curlerr)
				}
				var video SpaceVideo
				err := json.Unmarshal(body, &video)
				if err != nil {
					log.Error(err)
				}
				stat.Video += video.Data.Page.Count
			}
			defer wg.Done()
		}()
		wg.Wait()
		return stat
	} else {
		log.WithFields(log.Fields{
			"Vtuber": Data.ENName,
		}).Info("BiliBili Space nill")
		return stat
	}
}

func (Data Member) GetTwitterFollow() (int, error) {
	if Data.TwitterName != "" {
		profile, err := twitterscraper.GetProfile(Data.TwitterName)
		if err != nil {
			return 0, err
		}
		return profile.FollowersCount, nil
	} else {
		return 0, nil
	}
}

func (Data Member) BliBiliFace() (string, error) {
	if Data.BiliBiliID == 0 {
		return "", nil
	} else {
		var (
			Info Avatar
		)
		body, errcurl := network.CoolerCurl("https://api.bilibili.com/x/space/acc/info?mid="+strconv.Itoa(Data.BiliBiliID), BiliBiliSession)
		if errcurl != nil {
			return "", errcurl
		}
		err := json.Unmarshal(body, &Info)
		if err != nil {
			return "", errcurl
		}

		return strings.Replace(Info.Data.Face, "http", "https", -1), nil
	}
}

func (Data Member) GetTwitchAvatar() (string, error) {
	if Data.TwitchName != "" {
		resp, err := TwitchClient.GetUsers(&helix.UsersParams{
			Logins: []string{Data.TwitchName},
		})
		if err != nil {
			return "", err
		}
		for _, v := range resp.Data.Users {
			return v.ProfileImageURL, nil
		}
	}
	return "", nil
}

func CheckYoutube() {
	Data := database.GetGroups()
	for i := 0; i < len(Data); i++ {
		var wg sync.WaitGroup
		for i, NameData := range database.GetMembers(Data[i].ID) {
			wg.Add(1)
			go func(Name database.Member) {
				if Name.YoutubeID != "" {
					log.WithFields(log.Fields{
						"Vtube":        Name.EnName,
						"Youtube ID":   Name.YoutubeID,
						"Vtube Region": Name.Region,
					}).Info("Checking yt")
					FilterYt(Name, &wg)
				}

			}(NameData)
			if i%10 == 0 {
				wg.Wait()
			}
		}
		wg.Wait()
	}
}

func CheckTBiliBili() {
	DataGroup := database.GetGroups()
	for k := 0; k < len(DataGroup); k++ {
		DataMember := database.GetMembers(DataGroup[k].ID)
		for z := 0; z < len(DataMember); z++ {
			if DataMember[z].BiliBiliHashtags != "" {
				log.WithFields(log.Fields{
					"Group":  DataGroup[k].GroupName,
					"Vtuber": DataMember[z].EnName,
				}).Info("Start crawler T.bilibili")
				body, err := network.Curl("https://api.vc.bilibili.com/topic_svr/v1/topic_svr/topic_new?topic_name="+url.QueryEscape(DataMember[z].BiliBiliHashtags), nil)
				if err != nil {
					log.Error(err)
				}
				var (
					TB              TBiliBili
					DynamicIDStrTmp string
				)
				_ = json.Unmarshal(body, &TB)
				if (len(TB.Data.Cards) > 0) && TB.Data.Cards[0].Desc.DynamicIDStr != DynamicIDStrTmp {
					DynamicIDStrTmp = TB.Data.Cards[0].Desc.DynamicIDStr
					for i := 0; i < len(TB.Data.Cards); i++ {
						var (
							STB  SubTbili
							img  []string
							nope bool
						)
						_ = json.Unmarshal([]byte(TB.Data.Cards[i].Card), &STB)
						if STB.Item.Pictures != nil && TB.Data.Cards[i].Desc.Type == 2 { //type 2 is picture post (prob,heheheh)
							niggerlist := []string{"解锁专属粉丝卡片", "Official", "twitter.com", "咖啡厅", "CD", "专辑", "PIXIV", "遇", "marshmallow-qa.com"}
							for _, Nworld := range niggerlist {
								nope, _ = regexp.MatchString(Nworld, STB.Item.Description)
								if nope {
									break
								}
							}
							New := database.GetTBiliBili(TB.Data.Cards[i].Desc.DynamicIDStr)

							if New && !nope {
								log.WithFields(log.Fields{
									"Group":  DataGroup[k].GroupName,
									"Vtuber": DataMember[z].EnName,
								}).Info("New Fanart")
								for l := 0; l < len(STB.Item.Pictures); l++ {
									img = append(img, STB.Item.Pictures[l].ImgSrc)
								}

								Data := database.TBiliBili{
									URL:        "https://t.bilibili.com/" + TB.Data.Cards[i].Desc.DynamicIDStr + "?tab=2",
									Author:     TB.Data.Cards[i].Desc.UserProfile.Info.Uname,
									Avatar:     TB.Data.Cards[i].Desc.UserProfile.Info.Face,
									Like:       TB.Data.Cards[i].Desc.Like,
									Photos:     img,
									Dynamic_id: TB.Data.Cards[i].Desc.DynamicIDStr,
									Text:       STB.Item.Description,
									Member:     DataMember[z],
									Group:      DataGroup[k],
								}
								log.Info("Send to database")
								Data.InputTBiliBili()
							} else {
								log.WithFields(log.Fields{
									"Group":  DataGroup[k].GroupName,
									"Vtuber": DataMember[z].EnName,
								}).Info("Still same")
							}
						}
					}
				} else {
					log.WithFields(log.Fields{
						"Group":  DataGroup[k].GroupName,
						"Vtuber": DataMember[z].EnName,
					}).Info("Still same")
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func CheckLiveBiliBili() {
	log.Info("Start check BiliBili room")
	for _, Group := range database.GetGroups() {
		for _, Member := range database.GetMembers(Group.ID) {
			if Member.BiliBiliID != 0 {
				log.WithFields(log.Fields{
					"Group":   Group.GroupName,
					"SpaceID": Member.EnName,
				}).Info("Check Room")
				var (
					ScheduledStart time.Time
				)
				DataDB, err := database.GetRoomData(Member.ID, Member.BiliRoomID)
				if err != nil {
					log.Error(err)
				}
				Status, err := bilibili.GetRoomStatus(Member.BiliRoomID)
				if err != nil {
					log.Error(err)
				}
				loc, _ := time.LoadLocation("Asia/Shanghai")
				if Status.Data.RoomInfo.LiveStartTime != 0 {
					ScheduledStart = time.Unix(int64(Status.Data.RoomInfo.LiveStartTime), 0).In(loc)
				} else {
					ScheduledStart = time.Time{}
				}
				Data := map[string]interface{}{
					"LiveRoomID":     Member.BiliRoomID,
					"Status":         "",
					"Title":          Status.Data.RoomInfo.Title,
					"Thumbnail":      Status.Data.RoomInfo.Cover,
					"Description":    Status.Data.NewsInfo.Content,
					"PublishedAt":    time.Time{},
					"ScheduledStart": ScheduledStart,
					"Face":           Status.Data.AnchorInfo.BaseInfo.Face,
					"Online":         Status.Data.RoomInfo.Online,
					"BiliBiliID":     Member.BiliBiliID,
					"MemberID":       Member.ID,
				}
				if Status.CheckScheduleLive() {
					//Live
					log.WithFields(log.Fields{
						"Group":      Group.GroupName,
						"VtuberName": Member.Name,
					}).Info("Status Live")
					Data["Status"] = "Live"
					LiveBiliBili(Data)
				} else if !Status.CheckScheduleLive() && DataDB.Status == "Live" {
					//prob past
					log.WithFields(log.Fields{
						"Group":      Group.GroupName,
						"VtuberName": Member.Name,
					}).Info("Status Past")
					Data["Status"] = "Past"
					LiveBiliBili(Data)
				} else if DataDB.LiveRoomID == 0 {
					log.WithFields(log.Fields{
						"Group":      Group.GroupName,
						"VtuberName": Member.Name,
					}).Info("Status Unknown")
					Data["Status"] = "Unknown"
					LiveBiliBili(Data)
				}
			}
		}
	}
}

func CheckTwitch() {
	log.Info("Start check Twitch")
	for _, Group := range database.GetGroups() {
		for _, Member := range database.GetMembers(Group.ID) {
			if Member.TwitchName != "" {
				result, err := TwitchClient.GetStreams(&helix.StreamsParams{
					UserLogins: []string{Member.TwitchName},
				})
				if err != nil {
					log.Error(err)
				}
				if len(result.Data.Streams) > 0 {
					for _, Stream := range result.Data.Streams {
						if strings.ToLower(Stream.UserName) == strings.ToLower(Member.TwitchName) {
							GameResult, err := TwitchClient.GetGames(&helix.GamesParams{
								IDs: []string{Stream.GameID},
							})
							if err != nil {
								log.Error(err)
							}
							Stream.ThumbnailURL = strings.Replace(Stream.ThumbnailURL, "{width}", "1280", -1)
							Stream.ThumbnailURL = strings.Replace(Stream.ThumbnailURL, "{height}", "720", -1)
							log.WithFields(log.Fields{
								"Group":      Group.GroupName,
								"VtuberName": Member.Name,
								"Status":     Stream.Type,
							}).Info("Twitch status live")
							AddTwitchInfo(map[string]interface{}{
								"MemberID":       Member.ID,
								"Status":         Stream.Type,
								"Title":          Stream.Title,
								"Viewers":        Stream.ViewerCount,
								"ScheduledStart": Stream.StartedAt,
								"Thumbnails":     Stream.ThumbnailURL,
								"Game":           GameResult.Data.Games[0].Name,
								"MemberName":     Member.Name,
								"GroupName":      Group.GroupName,
							})
						}
					}
				} else {
					log.WithFields(log.Fields{
						"Group":      Group.GroupName,
						"VtuberName": Member.Name,
					}).Info("Twitch status nill")
					AddTwitchInfo(map[string]interface{}{
						"MemberID":       Member.ID,
						"Status":         "Past",
						"Title":          "",
						"Viewers":        0,
						"ScheduledStart": time.Time{},
						"Thumbnails":     "",
						"Game":           "",
						"MemberName":     Member.Name,
						"GroupName":      Group.GroupName,
					})
				}
			}
		}
	}
}

func CheckSpaceBiliBili() {
	Group := database.GetGroups()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	for z := 0; z < len(Group); z++ {
		Name := database.GetMembers(Group[z].ID)
		for k := 0; k < len(Name); k++ {
			if Name[k].BiliBiliID != 0 {
				log.WithFields(log.Fields{
					"Group":   Group[z].GroupName,
					"SpaceID": Name[k].EnName,
				}).Info("Check Space")
				var (
					PushVideo SpaceVideo
					videotype string
					url       []string
				)
				baseurl := "https://api.bilibili.com/x/space/arc/search?mid=" + strconv.Itoa(Name[k].BiliBiliID) + "&ps=100"
				url = []string{baseurl + "&tid=1", baseurl + "&tid=3", baseurl + "&tid=4"}
				for f := 0; f < len(url); f++ {
					body, err := network.Curl(url[f], nil)
					if err != nil {
						log.Error(err)
					}
					var tmp SpaceVideo
					err = json.Unmarshal(body, &tmp)
					if err != nil {
						log.Error(err)
					}
					for _, Vlist := range tmp.Data.List.Vlist {
						PushVideo.Data.List.Vlist = append(PushVideo.Data.List.Vlist, Vlist)
					}
				}

				for _, video := range PushVideo.Data.List.Vlist {
					if Cover, _ := regexp.MatchString("(?m)(cover|song|feat|music|翻唱|mv)", strings.ToLower(video.Title)); Cover {
						videotype = "Covering"
					} else {
						videotype = "Streaming"
					}
					tmp := database.InputBiliBili{
						VideoID:  video.Bvid,
						Type:     videotype,
						Title:    video.Title,
						Thum:     "https:" + video.Pic,
						Desc:     video.Description,
						Update:   time.Unix(int64(video.Created), 0).In(loc),
						Viewers:  video.Play,
						MemberID: Name[k].ID,
					}
					tmp.InputSpaceVideo()
				}
			}
		}
	}
}

type BiliStat struct {
	Follow BiliFollow
	Like   LikeView
	Video  int
}

type LikeView struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		Archive struct {
			View int `json:"view"`
		} `json:"archive"`
		Article struct {
			View int `json:"view"`
		} `json:"article"`
		Likes int `json:"likes"`
	} `json:"data"`
}

type BiliFollow struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		Mid       int `json:"mid"`
		Following int `json:"following"`
		Whisper   int `json:"whisper"`
		Black     int `json:"black"`
		Follower  int `json:"follower"`
	} `json:"data"`
}
