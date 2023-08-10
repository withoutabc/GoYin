package main

import (
	"GoYin/server/common/consts"
	"GoYin/server/kitex_gen/base"
	interaction "GoYin/server/kitex_gen/interaction"
	"GoYin/server/service/interaction/model"
	"context"
	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
	"strconv"
	"time"
)

// InteractionServerImpl implements the last service interface defined in the IDL.
type InteractionServerImpl struct {
	RedisManager
	MysqlManager
}
type MysqlManager interface {
	FavoriteAction(ctx context.Context, userId, videoId int64) error
	UnFavoriteAction(ctx context.Context, userId, videoId int64) error
	GetFavoriteVideoIdList(ctx context.Context, userId int64) ([]int64, error)
	Comment(ctx context.Context, comment *model.Comment) error
	DeleteComment(ctx context.Context, commentId int64) error
	GetComment(ctx context.Context, videoId int64) ([]*model.Comment, error)
	GetFavoriteCount(ctx context.Context, videoId int64) (int64, error)
	GetCommentCount(ctx context.Context, videoId int64) (int64, error)
	JudgeIsFavoriteCount(ctx context.Context, videoId, userId int64) (bool, error)
}
type RedisManager interface {
	FavoriteAction(ctx context.Context, userId, videoId int64) error
	UnFavoriteAction(ctx context.Context, userId, videoId int64) error
	GetFavoriteVideoIdList(ctx context.Context, userId int64) ([]int64, error)
	Comment(ctx context.Context, comment *model.Comment) error
	DeleteComment(ctx context.Context, commentId int64) error
	GetComment(ctx context.Context, videoId int64) ([]*model.Comment, error)
	GetFavoriteCount(ctx context.Context, videoId int64) (int64, error)
	GetCommentCount(ctx context.Context, videoId int64) (int64, error)
	JudgeIsFavoriteCount(ctx context.Context, videoId, userId int64) (bool, error)
}

// Favorite implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) Favorite(ctx context.Context, req *interaction.DouyinFavoriteActionRequest) (resp *interaction.DouyinFavoriteActionResponse, err error) {
	resp = new(interaction.DouyinFavoriteActionResponse)
	if req.ActionType == consts.Like {
		if err := s.MysqlManager.FavoriteAction(ctx, req.UserId, req.VideoId); err != nil {
			//回滚
			klog.Errorf("interaction mysql favorite failed,err", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction mysql favorite failed",
			}
			return resp, err
		}
		err = s.RedisManager.FavoriteAction(ctx, req.UserId, req.VideoId)
		if err != nil {
			klog.Errorf("interaction redis favorite failed,err", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction redis favorite failed",
			}
			return resp, err
		}
	} else if req.ActionType == consts.UnLike {
		if err := s.MysqlManager.UnFavoriteAction(ctx, req.UserId, req.VideoId); err != nil {
			//回滚
			klog.Errorf("interaction mysql unFavorite failed,err", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction mysql unFavorite failed",
			}
			return resp, err
		}
		err = s.RedisManager.UnFavoriteAction(ctx, req.UserId, req.VideoId)
		if err != nil {
			klog.Errorf("interaction redis unFavorite failed,err", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction redis unFavorite failed",
			}
			return resp, err
		}
	} else {
		resp.BaseResp = &base.DouyinBaseResponse{
			StatusCode: 500,
			StatusMsg:  "interaction invalid action type",
		}
		return resp, err
	}
	resp.BaseResp = &base.DouyinBaseResponse{
		StatusCode: 200,
		StatusMsg:  "interaction FavoriteAction success",
	}
	return resp, nil
}

// GetFavoriteVideoIdList implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) GetFavoriteVideoIdList(ctx context.Context, req *interaction.DouyinGetFavoriteVideoIdListRequest) (resp *interaction.DouyinGetFavoriteVideoIdListResponse, err error) {
	resp = new(interaction.DouyinGetFavoriteVideoIdListResponse)
	res, err := s.RedisManager.GetFavoriteVideoIdList(ctx, req.UserId)
	if err != nil {
		klog.Errorf("interaction redis get favorite video id list failed,", err)
		res, err = s.MysqlManager.GetFavoriteVideoIdList(ctx, req.UserId)
		if err != nil {
			klog.Errorf("interaction mysql get favorite video id list failed,", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction get favorite video id list failed",
			}
			return resp, err
		}
	}
	resp.VideoIdList = res
	resp.BaseResp = &base.DouyinBaseResponse{
		StatusCode: 200,
		StatusMsg:  "interaction get favorite video id list success",
	}
	return
}

// Comment implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) Comment(ctx context.Context, req *interaction.DouyinCommentActionRequest) (resp *interaction.DouyinCommentActionResponse, err error) {
	resp = new(interaction.DouyinCommentActionResponse)
	comment := &model.Comment{
		ID:          req.CommentId,
		UserId:      req.UserId,
		VideoId:     req.VideoId,
		ActionType:  req.ActionType,
		CommentText: req.CommentText,
		CreateDate:  time.Now().UnixNano(),
	}
	if req.ActionType == consts.Comment {
		sf, err := snowflake.NewNode(consts.CommentSnowFlakeNode)
		if err != nil {
			klog.Errorf("generate comment id failed: %s", err.Error())
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "generate comment id failed",
			}
		}
		comment.ID = sf.Generate().Int64()
	}

	if req.ActionType == consts.Comment {
		err = s.MysqlManager.Comment(ctx, comment)
		if err != nil {
			//回滚
			klog.Errorf("interaction mysql comment failed,", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction mysql comment failed",
			}
			return resp, err
		}
		err = s.RedisManager.Comment(ctx, comment)
		if err != nil {
			klog.Errorf("interaction redis comment failed,", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction redis comment failed",
			}
			return resp, err
		}
	} else if req.ActionType == consts.DeleteComment {
		err = s.MysqlManager.DeleteComment(ctx, req.CommentId)
		if err != nil {
			//回滚
			klog.Errorf("interaction mysql deleteComment failed,", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction mysql deleteComment failed",
			}
			return resp, err
		}
		err = s.RedisManager.DeleteComment(ctx, req.CommentId)
		if err != nil {
			klog.Errorf("interaction redis deleteComment failed,", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction redis deleteComment failed",
			}
			return resp, err
		}
	} else {
		resp.BaseResp = &base.DouyinBaseResponse{
			StatusCode: 500,
			StatusMsg:  "invalid action type",
		}
		return resp, nil
	}
	resp.BaseResp = &base.DouyinBaseResponse{
		StatusCode: 200,
		StatusMsg:  "interaction CommentAction success",
	}
	resp.Comment = &base.Comment{
		Id:         comment.ID,
		User:       &base.User{Id: req.UserId},
		Content:    comment.CommentText,
		CreateDate: strconv.FormatInt(comment.CreateDate, 10),
	}
	return resp, nil
}

// GetCommentList implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) GetCommentList(ctx context.Context, req *interaction.DouyinGetCommentListRequest) (resp *interaction.DouyinGetCommentListResponse, err error) {
	resp = new(interaction.DouyinGetCommentListResponse)

	commentList, err := s.RedisManager.GetComment(ctx, req.VideoId)
	if err != nil {
		klog.Errorf("interaction redis get commentList failed", err)
		commentList, err = s.MysqlManager.GetComment(ctx, req.VideoId)
		if err != nil {
			klog.Errorf("interaction mysql get commentList failed", err)
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction get commentList failed",
			}
			return resp, err
		}
	}
	for _, v := range commentList {
		resp.CommentList = append(resp.CommentList, &base.Comment{
			Id:         v.ID,
			User:       &base.User{Id: v.UserId},
			Content:    v.CommentText,
			CreateDate: strconv.FormatInt(v.CreateDate, 10),
		})
	}
	resp.BaseResp = &base.DouyinBaseResponse{
		StatusCode: 200,
		StatusMsg:  "interaction get comment success",
	}
	return resp, nil
}

// GetVideoInteractInfo implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) GetVideoInteractInfo(ctx context.Context, req *interaction.DouyinGetVideoInteractInfoRequest) (resp *interaction.DouyinGetVideoInteractInfoResponse, err error) {
	resp = new(interaction.DouyinGetVideoInteractInfoResponse)

	commentNum, favoriteNum, isFavorite, err := s.getVideoInfo(ctx, req.VideoId, req.ViewerId)
	if err != nil {
		klog.Errorf("interaction get video info failed")
		resp.BaseResp = &base.DouyinBaseResponse{
			StatusCode: 500,
			StatusMsg:  "interaction get video failed",
		}
		return resp, err
	}
	resp.InteractInfo = &base.VideoInteractInfo{
		FavoriteCount: favoriteNum,
		CommentCount:  commentNum,
		IsFavorite:    isFavorite,
	}
	resp.BaseResp = &base.DouyinBaseResponse{
		StatusCode: 200,
		StatusMsg:  "interaction get video info success",
	}
	return resp, nil
}

// BatchGetVideoInteractInfo implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) BatchGetVideoInteractInfo(ctx context.Context, req *interaction.DouyinBatchGetVideoInteractInfoRequest) (resp *interaction.DouyinBatchGetVideoInteractInfoResponse, err error) {
	resp = new(interaction.DouyinBatchGetVideoInteractInfoResponse)
	for _, v := range req.VideoIdList {
		commentNum, favoriteNum, isFavorite, err := s.getVideoInfo(ctx, v, req.ViewerId)
		if err != nil {
			klog.Errorf("interaction get video info failed")
			resp.BaseResp = &base.DouyinBaseResponse{
				StatusCode: 500,
				StatusMsg:  "interaction get video failed",
			}
			return resp, err
		}
		resp.InteractInfoList = append(resp.InteractInfoList, &base.VideoInteractInfo{
			FavoriteCount: favoriteNum,
			CommentCount:  commentNum,
			IsFavorite:    isFavorite,
		})
	}
	resp.BaseResp = &base.DouyinBaseResponse{
		StatusCode: 200,
		StatusMsg:  "interaction get video info success",
	}

	return resp, nil
}
func (s *InteractionServerImpl) getVideoInfo(ctx context.Context, videoId, userId int64) (CommentNum, FavoriteNum int64, IsFavorite bool, err error) {
	CommentNum, err = s.RedisManager.GetCommentCount(ctx, videoId)
	if err != nil {
		klog.Errorf("interaction get video comment num failed,", err)
		CommentNum, err = s.MysqlManager.GetCommentCount(ctx, videoId)
		if err != nil {
			klog.Errorf("interaction get video comment num failed,", err)
			return CommentNum, FavoriteNum, IsFavorite, err
		}
	}
	FavoriteNum, err = s.RedisManager.GetFavoriteCount(ctx, videoId)
	if err != nil {
		klog.Errorf("interaction get video favorite num failed,", err)
		CommentNum, err = s.MysqlManager.GetFavoriteCount(ctx, videoId)
		if err != nil {
			klog.Errorf("interaction get favorite num failed,", err)
			return CommentNum, FavoriteNum, IsFavorite, err
		}
	}
	IsFavorite, err = s.RedisManager.JudgeIsFavoriteCount(ctx, videoId, userId)
	if err != nil {
		klog.Errorf("interaction judge isFavorite failed,", err)
		IsFavorite, err = s.MysqlManager.JudgeIsFavoriteCount(ctx, videoId, userId)
		if err != nil {
			klog.Errorf("interaction judge isFavorite failed,", err)
			return CommentNum, FavoriteNum, IsFavorite, err
		}
	}
	return CommentNum, FavoriteNum, IsFavorite, nil
}

// GetUserInteractInfo implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) GetUserInteractInfo(ctx context.Context, req *interaction.DouyinGetUserInteractInfoRequest) (resp *interaction.DouyinGetUserInteractInfoResponse, err error) {
	// TODO: Your code here...
	return
}

// BatchGetUserInteractInfo implements the InteractionServerImpl interface.
func (s *InteractionServerImpl) BatchGetUserInteractInfo(ctx context.Context, req *interaction.DouyinBatchGetUserInteractInfoRequest) (resp *interaction.DouyinBatchGetUserInteractInfoResponse, err error) {
	// TODO: Your code here...
	return
}
