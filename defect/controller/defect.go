package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/opensourceways/server-common-lib/controller"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/defect-manager/defect/app"
)

type DefectController struct {
	service app.DefectService
}

func AddRouteForDefectController(r *gin.RouterGroup, s app.DefectService) {
	ctl := DefectController{
		service: s,
	}

	r.GET("/v1/defect", ctl.Collect)
	r.POST("/v1/defect/bulletin", ctl.GenerateBulletin)
}

// Collect
// @Summary collect information of some defects
// @Description collect information of some defects
// @Tags  Defect
// @Accept json
// @Param	version  query string	 true	"collect defects of the version"
// @Success 200 {object} []app.CollectDefectsDTO
// @Failure 400 {object} string
// @Router /v1/defect [get]
func (ctl DefectController) Collect(ctx *gin.Context) {
	version := ctx.Query("version")

	if v, err := ctl.service.CollectDefects(version); err != nil {
		controller.SendFailedResp(ctx, "", err)
	} else {
		controller.SendRespOfGet(ctx, v)
	}
}

// GenerateBulletin
// @Summary generate security bulletin for some defects
// @Description generate security bulletin for some defects
// @Tags  Defect
// @Accept json
// @Param	param  body	 bulletinRequest	 true	"body of some issue number"
// @Success 201 {object} string
// @Failure 400 {object} string
// @Router /v1/defect/bulletin [post]
func (ctl DefectController) GenerateBulletin(ctx *gin.Context) {
	var req bulletinRequest
	if err := ctx.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		controller.SendBadRequestBody(ctx, err)

		return
	}

	go func() {
		logrus.Infof("generate bulletin processing of %v", req.IssueNumber)

		if err := ctl.service.GenerateBulletins(req.IssueNumber); err != nil {
			logrus.Errorf("generate bulletin of %v err: %s", req.IssueNumber, err.Error())
		} else {
			logrus.Infof("generate bulletin success of %v", req.IssueNumber)
		}
	}()

	controller.SendRespOfPost(ctx, "Processing: Data is being prepared, please wait patiently\n")
}
