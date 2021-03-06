package space

import (
	"strconv"
	"sync"
	"time"

	"github.com/JustHumanz/Go-Simp/pkg/config"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"

	database "github.com/JustHumanz/Go-Simp/pkg/database"
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
	VtubersData = c
	b.AddFunc(config.BiliBiliSpace, CheckSpaceVideo)
	log.Info("Enable space bilibili module")
}

func CheckSpaceVideo() {
	for _, GroupData := range VtubersData.VtuberData {
		if GroupData.GroupName != "Hololive" {
			wg := new(sync.WaitGroup)
			for i, MemberData := range GroupData.Members {
				wg.Add(1)
				go func(Group database.Group, Member database.Member, wg *sync.WaitGroup) {
					defer wg.Done()
					if Member.BiliBiliID != 0 {
						log.WithFields(log.Fields{
							"Group":      Group.GroupName,
							"Vtuber":     Member.EnName,
							"BiliBiliID": Member.BiliBiliID,
						}).Info("Checking Space BiliBili")

						if Group.GroupName == "Independen" {
							Group.IconURL = ""
						}

						Data := &CheckSctruct{
							Member: Member,
							Group:  Group,
						}
						Data.Check(strconv.Itoa(configfile.LimitConf.SpaceBiliBili)).SendNude()

					}
				}(GroupData, MemberData, wg)
				if i%config.Waiting == 0 {
					wg.Wait()
				}
			}
			wg.Wait()
		}
	}
}
