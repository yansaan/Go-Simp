const model = require("../models/model.js");

exports.memberAll = (_,res) => {
    console.log(res.params)
    model.GetMemberAll((err, data) => {
    if (err)
    res.status(500).send({
        message:
        err.message || "Some error."
    });
    else res.send(data);
    });
};

exports.memberName = (req, res) => {
    model.GetMemberName(req.params.name.split(","), (err, data) => {
    if (err) {
    if (err.kind === "not_found") {
        res.status(404).send({
        message: `Not found VtuberName with name ${req.params.name}.`
        });
    } else {
        res.status(500).send({
        message: "Error retrieving VtuberName with name " + req.params.name
        });
    }
    } else res.send(data);
});
};
  

exports.groupAll = (_,res) => {
    model.GetGroupAll((err, data) => {
        if (err)
        res.status(500).send({
            message:
            err.message || "Some error."
        });
        else res.send(data);
    });
};
    
exports.groupName = (req, res) => {
    model.GetGroupName(req.params.group.split(","), (err, data) => {
        if (err) {
        if (err.kind === "not_found") {
            res.status(404).send({
            message: `Not found GroupName with name ${req.params.group}.`
            });
        } else {
            res.status(500).send({
            message: "Error retrieving GroupName with name " + req.params.group
            });
        }
        } else res.send(data);
    });
};


exports.ytlivestream = (req, res) => {
    const Limit =  req.query.limit || 30
    if (Limit >= 100 ){
        res.status(401).send({
            message: `out of limit`
        });
        return
    } 
    model.GetYtLivestream(req.params.nog,req.params.status, Limit,(err, data) => {
        if (err) {
        if (err.kind === "not_found") {
            res.status(404).send({
            message: `It looks like ${req.params.nog} doesn't have a ${req.params.status} stream right now .`
            });
        } else {
            res.status(500).send({
            message: "Error retrieving GetYtLivestream with name " + req.params.nog
            });
        }
        } else res.send(data);
    });
};

exports.twitterd = (req, res) => {
    const Limit =  req.query.limit || 30
    if (Limit >= 300 ){
        res.status(401).send({
            message: `out of limit`
        });
        return
    }
    model.GetTwitter(req.params.nog, Limit,(err, data) => {
        if (err) {
        if (err.kind === "not_found") {
            res.status(404).send({
            message: `Not found GetTwitter with name ${req.params.nog}.`
            });
        } else {
            res.status(500).send({
            message: "Error retrieving GetTwitter with name " + req.params.nog
            });
        }
        } else res.send(data);
    });
};

exports.tBilibili = (req, res) => {
    const Limit =  req.query.limit || 30
    if (Limit >= 300 ){
        res.status(401).send({
            message: `out of limit`
        });
        return
    }
    model.GetTBilibili(req.params.nog, Limit,(err, data) => {
        if (err) {
        if (err.kind === "not_found") {
            res.status(404).send({
            message: `Not found GetTBilibili with name ${req.params.nog}.`
            });
        } else {
            res.status(500).send({
            message: "GetTBilibili Error LMAO " + req.params.nog
            });
        }
        } else res.send(data);
    });
};


exports.liveBilibili = (req, res) => {
    const Limit =  req.query.limit || 10
    if (Limit >= 30 ){
        res.status(200).send({
            message: `out of limit`
        });
        return
    }
    model.GetLiveBilibili(req.params.nog,req.params.status, Limit,(err, data) => {
        if (err) {
        if (err.kind === "not_found") {
            res.status(404).send({
            message: `It looks like ${req.params.nog} doesn't have a ${req.params.status} stream right now .`
            });
        } else {
            res.status(500).send({
            message: "Error retrieving GetYtLivestream with name " + req.params.nog
            });
        }
        } else res.send(data);
    });
};

exports.spaceBilibili = (req, res) => {
    const Limit =  req.query.limit || 30
    if (Limit >= 30 ){
        res.status(200).send({
            message: `out of limit`
        });
        return
    }
    model.GetSpaceBiliBIli(req.params.nog, Limit,(err, data) => {
        if (err) {
        if (err.kind === "not_found") {
            res.status(404).send({
            message: `It looks like ${req.params.nog} doesn't have a ${req.params.status} stream right now .`
            });
        } else {
            res.status(500).send({
            message: "Error retrieving GetYtLivestream with name " + req.params.nog
            });
        }
        } else res.send(data);
    });
};


exports.subscriber = (req, res) => {
    model.Getsubscriber(req.params.nog,(err, data) => {
        if (err) {
        if (err.kind === "not_found") {
            res.status(404).send({
            message: `It looks like ${req.params.nog} doesn't have a ${req.params.status} stream right now .`
            });
        } else {
            res.status(500).send({
            message: "Error retrieving GetYtLivestream with name " + req.params.nog
            });
        }
        } else res.send(data);
    });
};