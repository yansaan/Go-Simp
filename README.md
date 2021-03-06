# Vtuber DiscordBot

![alt text](https://raw.githubusercontent.com/JustHumanz/Go-Simp/master/Img/go-simp.png "Go-Simp")  
##### [Original Source](https://twitter.com/any_star_/status/1288184424320790528)
![Inline docs](https://github.com/JustHumanz/Go-Simp/workflows/Go-Simp/badge.svg)
----

## Introduction
A simple VTuber bot to serve notification Fanart(Twitter,Bilibili) and Livestream(Youtube,Bilibili)  
for setup see at [Guide](https://github.com/JustHumanz/Go-Simp/blob/master/Guide.md)

## Simple Setup
##### Livestream notification
First create a channel for this bot,Example `holosimps_channel`  
then execute `vtbot>enable live hololive` command,after that you can wait until hololive member going live and got notification  
And if you want ping some roles you can execute `vtbot>tag roles <Rolename> hololive` In other case if you want to remind before livestream started you can add `-setreminder {Minutes}`  
Example : `vtbot>tag roles @holosimp hololive -setreminder 20`

##### Fanart notification
First create a channel for this bot,Example `nijisimps`  
then execute `vtbot>enable art nijisanji` command,after that you can wait until new fanart from twitter appear

## Current notification support
See at [Web](https://go-simp.human-z.tech)

### Command
See at [Exec](https://go-simp.human-z.tech/Exec/)


### Notification&Command 


#### Add User to tag list
<p align="center">
  <img src="https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/AddUser.png" alt="New Upcoming live stream"/>
</p>

#### New Upcoming live stream  
<p align="center">
  <img src="https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/New%20Upcoming.png" alt="New Upcoming live stream"/>
</p>


#### Reminder  
Reminder 30 minutes before livestream start
<p align="center">
  <img src="https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/Reminder.png" alt="Reminder"/>
</p>


##### Upcoming command
<p align="center">
  <img src="https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/Youtube%20Upcoming.png" alt="Upcoming command"/>
</p>


##### New Fanart
<p align="center">
  <img src="https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/New%20Fanart.png" alt="New fanart"/>
</p>


##### Subscriber count
<p align="center">
  <img src="https://raw.githubusercontent.com/JustHumanz/Go-simp/master/Img/Subscount.png" alt="Subscriber count"/>
</p>



[Invite link](https://top.gg/bot/721964514018590802)

## TODO
- Add hall of flame for contributor
- Add twitch
- ~~Specially for Independent fan art will not be sent if there no one user/role tagged~~ (Done)
- ~~Add -liveonly -newupcoming and -rm_liveonly -rm_newupcoming~~ (Done)
- ~~Add ArkNET~~  (Done)
- Add web interface (On progress)
- ~~Add customize reminder~~ (Done)
- Vtuber collaboration Detection (Pending)
- ~~Make a form for add New Vtuber~~ (Done)
- ~~Move to microservice~~ (Done)



## CONTRIBUTING
if you can code you can review my shitty code and make it better or fix my bot command (i know my command bot is `too technical`)  

Very open for Report bug or suggestion feature and emoji 

