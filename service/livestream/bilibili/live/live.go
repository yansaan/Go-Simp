package live

import (
	"regexp"
	"strconv"
	"sync"
	"time"

	config "github.com/JustHumanz/Go-Simp/pkg/config"
	database "github.com/JustHumanz/Go-Simp/pkg/database"
	engine "github.com/JustHumanz/Go-Simp/pkg/engine"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

var (
	loc         *time.Location
	Bot         *discordgo.Session
	VtubersData database.VtubersPayload
	configfile  config.ConfigFile
)

//Start start twitter module
func Start(a *discordgo.Session, b *cron.Cron, c database.VtubersPayload, d config.ConfigFile) {
	loc, _ = time.LoadLocation("Asia/Shanghai") /*Use CST*/
	Bot = a
	configfile = d
	b.AddFunc(config.BiliBiliLive, CheckLiveSchedule)
	VtubersData = c
	log.Info("Enable Live BiliBili module")
}

func CheckLiveSchedule() {
	for _, GroupData := range VtubersData.VtuberData {
		var wg sync.WaitGroup
		for i, MemberData := range GroupData.Members {
			wg.Add(1)
			log.WithFields(log.Fields{
				"Group":  GroupData.GroupName,
				"Vtuber": MemberData.EnName,
			}).Info("Checking LiveBiliBili")
			go CheckBili(GroupData, MemberData, &wg)
			if i%10 == 0 {
				wg.Wait()
			}
		}
	}
}

func CheckBili(Group database.Group, Member database.Member, wg *sync.WaitGroup) {
	defer wg.Done()
	if Member.BiliBiliID != 0 {
		var (
			ScheduledStart time.Time
			Data           LiveBili
		)
		DataDB, err := database.GetRoomData(Member.ID, Member.BiliRoomID)
		if err != nil {
			log.Error(err)
		}
		Status, err := GetRoomStatus(Member.BiliRoomID)
		if err != nil {
			log.Error(err)
		}

		if DataDB != nil {
			Data.AddData(DataDB)
			if Status.CheckScheduleLive() && DataDB.Status != "Live" {
				//Live
				if Status.Data.RoomInfo.LiveStartTime != 0 {
					ScheduledStart = time.Unix(int64(Status.Data.RoomInfo.LiveStartTime), 0).In(loc)
				} else {
					ScheduledStart = time.Now().In(loc)
				}

				if match, _ := regexp.MatchString("404.jpg", Group.IconURL); match {
					Group.IconURL = ""
				}

				log.WithFields(log.Fields{
					"Group":  Group.GroupName,
					"Vtuber": Member.EnName,
					"Start":  ScheduledStart,
				}).Info("Start live right now")

				Data.SetStatus("Live").
					UpdateSchdule(ScheduledStart).
					UpdateOnline(Status.Data.RoomInfo.Online).
					SetMember(Member).
					SetGroup(Group)

				err = Data.Crotttt()
				if err != nil {
					log.Error(err)
				}

				Data.RoomData.UpdateLiveBili(Member.ID)

			} else if !Status.CheckScheduleLive() && DataDB.Status == "Live" {
				//prob past
				log.WithFields(log.Fields{
					"Group":  Group.GroupName,
					"Vtuber": Member.EnName,
					"Start":  ScheduledStart,
				}).Info("Past live stream")
				err := engine.RemoveEmbed(strconv.Itoa(DataDB.LiveRoomID), Bot)
				if err != nil {
					log.Error(err)
				}
				Data.SetStatus("Past").
					UpdateOnline(Status.Data.RoomInfo.Online)

				Data.RoomData.UpdateLiveBili(Member.ID)
			} else {
				//update online
				Data.UpdateOnline(Status.Data.RoomInfo.Online)
				Data.RoomData.UpdateLiveBili(Member.ID)
			}
		}

	}
}
