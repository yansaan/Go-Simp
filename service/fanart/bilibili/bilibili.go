package bilibili

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	config "github.com/JustHumanz/Go-Simp/pkg/config"
	database "github.com/JustHumanz/Go-Simp/pkg/database"
	network "github.com/JustHumanz/Go-Simp/pkg/network"
)

//Public variable
var (
	Bot         *discordgo.Session
	VtubersData database.VtubersPayload
	configfile  config.ConfigFile
)

//Start start twitter module
func Start(a *discordgo.Session, b *cron.Cron, c database.VtubersPayload, d config.ConfigFile) {
	Bot = a
	VtubersData = c
	configfile = d
	b.AddFunc(config.BiliBiliFanart, CheckNew)
	log.Info("Enable bilibili fanart module")
}

//CheckNew Start Check new fanart
func CheckNew() {
	for _, Group := range VtubersData.VtuberData {
		for _, Member := range Group.Members {
			if Member.BiliBiliHashtags != "" {
				log.WithFields(log.Fields{
					"Group":  Group.GroupName,
					"Vtuber": Member.EnName,
				}).Info("Start crawler bilibili")
				body, errcurl := network.CoolerCurl("https://api.vc.bilibili.com/topic_svr/v1/topic_svr/topic_new?topic_name="+url.QueryEscape(Member.BiliBiliHashtags), nil)
				if errcurl != nil {
					log.Error(errcurl)
				}
				var (
					TB  TBiliBili
					ctx = context.Background()
					rdb = database.FanartCache
				)
				_ = json.Unmarshal(body, &TB)
				if len(TB.Data.Cards) > 0 {
					for i := 0; i < len(TB.Data.Cards); i++ {
						var (
							STB SubTbili
							img []string
						)
						err := json.Unmarshal([]byte(TB.Data.Cards[i].Card), &STB)
						if err != nil {
							log.Error(err)
						}
						if STB.Item.Pictures != nil && TB.Data.Cards[i].Desc.Type == 2 { //type 2 is picture post (prob,heheheh)
							_, err := rdb.Get(ctx, TB.Data.Cards[i].Desc.DynamicIDStr).Result()
							if err == redis.Nil {
								New := database.GetTBiliBili(TB.Data.Cards[i].Desc.DynamicIDStr)
								if New {
									if len(STB.Item.Pictures) > 0 {
										if STB.Item.Pictures[0].ImgSrc != "" {
											log.WithFields(log.Fields{
												"Vtuber": Member.EnName,
												"Img":    STB.Item.Pictures[0].ImgSrc,
											}).Info("New Fanart")
											for l := 0; l < len(STB.Item.Pictures); l++ {
												img = append(img, STB.Item.Pictures[l].ImgSrc)
											}

											log.WithFields(log.Fields{"Group": Group.GroupName, "Vtuber": Member.EnName}).Info("Push to notif")
											Data := database.TBiliBili{
												URL:        "https://t.bilibili.com/" + TB.Data.Cards[i].Desc.DynamicIDStr + "?tab=2",
												Author:     TB.Data.Cards[i].Desc.UserProfile.Info.Uname,
												Avatar:     TB.Data.Cards[i].Desc.UserProfile.Info.Face,
												Like:       TB.Data.Cards[i].Desc.Like,
												Photos:     img,
												Dynamic_id: TB.Data.Cards[i].Desc.DynamicIDStr,
												Text:       STB.Item.Description,
												Member:     Member,
												Group:      Group,
											}
											err := Data.InputTBiliBili()
											if err != nil {
												log.Error(err)
											} else {
												err := PushNotif(Data)
												if err != nil {
													log.Error(err)
												}
											}
										}
										err = rdb.Set(ctx, TB.Data.Cards[i].Desc.DynamicIDStr, Member.Name, 30*time.Minute).Err()
										if err != nil {
											log.Error(err)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

/*
//Mirroring fanart to imgur *sometime discord fail to load image because bilibili CDN(prob)*
func (Data SubTbili) Mirroring() (string, int, error) {
	link := Data.Item.Pictures[0].ImgSrc
		buf := &bytes.Buffer{}
		writer := multipart.NewWriter(buf)
		err := writer.WriteField("image", link)
		err = writer.WriteField("title", Data.Item.Title)
		err = writer.WriteField("name", Data.User.Name)
		err = writer.Close()
		if err != nil {
			log.Error(err)
		}
	color, err := engine.GetColor(config.TmpDir, link)
	if err != nil {
		log.Error(err)
	}
		req, err := http.NewRequest("POST", "https://api.imgur.com/3/image", buf)
		if err != nil {
			return "", 0, err
		}

		req.Header.Set("Authorization", "Client-ID "+config.ImgurClient)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Votre) AppleWebKit/601.2 (KHTML, like Gecko)")
		req.Header.Set("Content-Type", writer.FormDataContentType())

		htt := http.Client{Timeout: time.Second * 20}
		res, err := htt.Do(req)
		if err != nil {
			log.Error(err)
			log.Info("bypass it")
			return link, color, nil
		}

		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Error(err)
			log.Info("bypass it")
			return link, color, nil
		}

		if err != nil || res.StatusCode != 200 {
			log.Error(err, res.Status, string(b))
			log.Info("bypass it")
			return link, color, nil
		}

		var tmp Imgur
		err = json.Unmarshal(b, &tmp)
		if err != nil {
			log.Error(err)
			return "", 0, err
		}

		color, err = engine.GetColor("/tmp/bilibili", tmp.Data.Link)
		if err != nil {
			log.Error(err, " ", link)
			return "", 0, err
		}

	return link, color, nil
}
*/

//Imgur struct
type Imgur struct {
	Data struct {
		ID          string        `json:"id"`
		Title       interface{}   `json:"title"`
		Description interface{}   `json:"description"`
		Datetime    int           `json:"datetime"`
		Type        string        `json:"type"`
		Animated    bool          `json:"animated"`
		Width       int           `json:"width"`
		Height      int           `json:"height"`
		Size        int           `json:"size"`
		Views       int           `json:"views"`
		Bandwidth   int           `json:"bandwidth"`
		Vote        interface{}   `json:"vote"`
		Favorite    bool          `json:"favorite"`
		Nsfw        interface{}   `json:"nsfw"`
		Section     interface{}   `json:"section"`
		AccountURL  interface{}   `json:"account_url"`
		AccountID   int           `json:"account_id"`
		IsAd        bool          `json:"is_ad"`
		InMostViral bool          `json:"in_most_viral"`
		Tags        []interface{} `json:"tags"`
		AdType      int           `json:"ad_type"`
		AdURL       string        `json:"ad_url"`
		InGallery   bool          `json:"in_gallery"`
		Deletehash  string        `json:"deletehash"`
		Name        string        `json:"name"`
		Link        string        `json:"link"`
	} `json:"data"`
	Success bool `json:"success"`
	Status  int  `json:"status"`
}
