package main

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

const (
	MaxLoop = 2500
)

type StateType int32

const (
	ST_S    StateType     = 1 << iota // 1
	ST_C                              // 2
	ST_A                              // 4
	ST_L                              // 8
	ST_N                              // 16
	ST_U                              // 32
	ST_MASK = 1<<iota - 1             // (1 << 6) -1 (63)
	ST_SCA  = ST_S | ST_C | ST_A
)

type AttackType int32

const (
	AT_NA AttackType = 1 << (iota + 6)
	AT_NT
	AT_NP
	AT_SA
	AT_ST
	AT_SP
	AT_HA
	AT_HT
	AT_HP
	AT_AA  = AT_NA | AT_SA | AT_HA
	AT_AT  = AT_NT | AT_ST | AT_HT
	AT_AP  = AT_NP | AT_SP | AT_HP
	AT_ALL = AT_AA | AT_AT | AT_AP
	AT_AN  = AT_NA | AT_NT | AT_NP
	AT_AS  = AT_SA | AT_ST | AT_SP
	AT_AH  = AT_HA | AT_HT | AT_HP
)

type MoveType int32

const (
	MT_I MoveType = 1 << (iota + 15)
	MT_H
	MT_A
	MT_U
)

type HitFlag int32

const (
	HF_H HitFlag = 1 << iota
	HF_L
	HF_A
	HF_D
	HF_F
	HF_P
	HF_MNS
	HF_PLS
	HF_M = HF_H | HF_L
)

type ValueType int

const (
	VT_None ValueType = iota
	VT_Float
	VT_Int
	VT_Bool
	VT_SFalse // Undefined
)

type OpCode byte

const (
	OC_var OpCode = iota
	OC_sysvar
	OC_fvar
	OC_sysfvar
	OC_localvar
	OC_int8
	OC_int
	OC_int64
	OC_float
	OC_pop
	OC_dup
	OC_swap
	OC_run
	OC_nordrun
	OC_jsf8
	OC_jmp8
	OC_jz8
	OC_jnz8
	OC_jmp
	OC_jz
	OC_jnz
	OC_eq
	OC_ne
	OC_gt
	OC_ge
	OC_lt
	OC_le
	OC_neg
	OC_blnot
	OC_bland
	OC_blxor
	OC_blor
	OC_not
	OC_and
	OC_xor
	OC_or
	OC_add
	OC_sub
	OC_mul
	OC_div
	OC_mod
	OC_pow
	OC_abs
	OC_exp
	OC_ln
	OC_log
	OC_cos
	OC_sin
	OC_tan
	OC_acos
	OC_asin
	OC_atan
	OC_floor
	OC_ceil
	OC_ifelse
	OC_time
	OC_animtime
	OC_animelemtime
	OC_animelemno
	OC_statetype
	OC_movetype
	OC_ctrl
	OC_command
	OC_random
	OC_pos_x
	OC_pos_y
	OC_vel_x
	OC_vel_y
	OC_vel_z
	OC_screenpos_x
	OC_screenpos_y
	OC_facing
	OC_anim
	OC_animexist
	OC_selfanimexist
	OC_alive
	OC_life
	OC_lifemax
	OC_power
	OC_powermax
	OC_canrecover
	OC_roundstate
	OC_roundswon
	OC_ishelper
	OC_numhelper
	OC_numexplod
	OC_numprojid
	OC_numproj
	OC_numtext
	OC_teammode
	OC_teamside
	OC_hitdefattr
	OC_inguarddist
	OC_movecontact
	OC_movehit
	OC_moveguarded
	OC_movereversed
	OC_projcontacttime
	OC_projhittime
	OC_projguardedtime
	OC_projcanceltime
	OC_backedge
	OC_backedgedist
	OC_backedgebodydist
	OC_frontedge
	OC_frontedgedist
	OC_frontedgebodydist
	OC_leftedge
	OC_rightedge
	OC_topedge
	OC_bottomedge
	OC_camerapos_x
	OC_camerapos_y
	OC_camerazoom
	OC_gamewidth
	OC_gameheight
	OC_screenwidth
	OC_screenheight
	OC_stateno
	OC_prevstateno
	OC_id
	OC_playeridexist
	OC_gametime
	OC_numtarget
	OC_numenemy
	OC_numpartner
	OC_ailevel
	OC_palno
	OC_hitcount
	OC_uniqhitcount
	OC_hitpausetime
	OC_hitover
	OC_hitshakeover
	OC_hitfall
	OC_hitvel_x
	OC_hitvel_y
	OC_hitvel_z
	OC_player
	OC_parent
	OC_root
	OC_helper
	OC_target
	OC_partner
	OC_enemy
	OC_enemynear
	OC_playerid
	OC_playerindex
	OC_helperindex
	OC_p2
	OC_stateowner
	OC_rdreset
	OC_const_
	OC_st_
	OC_ex_
	OC_ex2_
)
const (
	OC_const_data_life OpCode = iota
	OC_const_data_power
	OC_const_data_guardpoints
	OC_const_data_dizzypoints
	OC_const_data_attack
	OC_const_data_defence
	OC_const_data_fall_defence_up
	OC_const_data_fall_defence_mul
	OC_const_data_liedown_time
	OC_const_data_airjuggle
	OC_const_data_sparkno
	OC_const_data_guard_sparkno
	OC_const_data_hitsound_channel
	OC_const_data_guardsound_channel
	OC_const_data_ko_echo
	OC_const_data_volume
	OC_const_data_intpersistindex
	OC_const_data_floatpersistindex
	OC_const_size_xscale
	OC_const_size_yscale
	OC_const_size_ground_back
	OC_const_size_ground_front
	OC_const_size_air_back
	OC_const_size_air_front
	OC_const_size_height_stand
	OC_const_size_height_crouch
	OC_const_size_height_air_top
	OC_const_size_height_air_bottom
	OC_const_size_height_down
	OC_const_size_attack_dist_width_front
	OC_const_size_attack_dist_width_back
	OC_const_size_attack_dist_height_top
	OC_const_size_attack_dist_height_bottom
	OC_const_size_attack_dist_depth_top
	OC_const_size_attack_dist_depth_bottom
	OC_const_size_attack_depth_top
	OC_const_size_attack_depth_bottom
	OC_const_size_proj_attack_dist_width_front
	OC_const_size_proj_attack_dist_width_back
	OC_const_size_proj_attack_dist_height_top
	OC_const_size_proj_attack_dist_height_bottom
	OC_const_size_proj_attack_dist_depth_top
	OC_const_size_proj_attack_dist_depth_bottom
	OC_const_size_proj_doscale
	OC_const_size_head_pos_x
	OC_const_size_head_pos_y
	OC_const_size_mid_pos_x
	OC_const_size_mid_pos_y
	OC_const_size_shadowoffset
	OC_const_size_draw_offset_x
	OC_const_size_draw_offset_y
	OC_const_size_depth_top
	OC_const_size_depth_bottom
	OC_const_size_weight
	OC_const_size_pushfactor
	OC_const_velocity_air_gethit_airrecover_add_x
	OC_const_velocity_air_gethit_airrecover_add_y
	OC_const_velocity_air_gethit_airrecover_back
	OC_const_velocity_air_gethit_airrecover_down
	OC_const_velocity_air_gethit_airrecover_fwd
	OC_const_velocity_air_gethit_airrecover_mul_x
	OC_const_velocity_air_gethit_airrecover_mul_y
	OC_const_velocity_air_gethit_airrecover_up
	OC_const_velocity_air_gethit_groundrecover_x
	OC_const_velocity_air_gethit_groundrecover_y
	OC_const_velocity_air_gethit_ko_add_x
	OC_const_velocity_air_gethit_ko_add_y
	OC_const_velocity_air_gethit_ko_ymin
	OC_const_velocity_airjump_back_x
	OC_const_velocity_airjump_down_x
	OC_const_velocity_airjump_down_y
	OC_const_velocity_airjump_down_z
	OC_const_velocity_airjump_fwd_x
	OC_const_velocity_airjump_neu_x
	OC_const_velocity_airjump_up_x
	OC_const_velocity_airjump_up_y
	OC_const_velocity_airjump_up_z
	OC_const_velocity_airjump_y
	OC_const_velocity_ground_gethit_ko_add_x
	OC_const_velocity_ground_gethit_ko_add_y
	OC_const_velocity_ground_gethit_ko_xmul
	OC_const_velocity_ground_gethit_ko_ymin
	OC_const_velocity_jump_back_x
	OC_const_velocity_jump_down_x
	OC_const_velocity_jump_down_y
	OC_const_velocity_jump_down_z
	OC_const_velocity_jump_fwd_x
	OC_const_velocity_jump_neu_x
	OC_const_velocity_jump_up_x
	OC_const_velocity_jump_up_y
	OC_const_velocity_jump_up_z
	OC_const_velocity_jump_y
	OC_const_velocity_run_back_x
	OC_const_velocity_run_back_y
	OC_const_velocity_run_down_x
	OC_const_velocity_run_down_y
	OC_const_velocity_run_down_z
	OC_const_velocity_run_fwd_x
	OC_const_velocity_run_fwd_y
	OC_const_velocity_run_up_x
	OC_const_velocity_run_up_y
	OC_const_velocity_run_up_z
	OC_const_velocity_runjump_back_x
	OC_const_velocity_runjump_back_y
	OC_const_velocity_runjump_down_x
	OC_const_velocity_runjump_down_y
	OC_const_velocity_runjump_down_z
	OC_const_velocity_runjump_fwd_x
	OC_const_velocity_runjump_up_x
	OC_const_velocity_runjump_up_y
	OC_const_velocity_runjump_up_z
	OC_const_velocity_runjump_y
	OC_const_velocity_walk_back_x
	OC_const_velocity_walk_down_x
	OC_const_velocity_walk_down_y
	OC_const_velocity_walk_down_z
	OC_const_velocity_walk_fwd_x
	OC_const_velocity_walk_up_x
	OC_const_velocity_walk_up_y
	OC_const_velocity_walk_up_z
	OC_const_movement_airjump_num
	OC_const_movement_airjump_height
	OC_const_movement_yaccel
	OC_const_movement_stand_friction
	OC_const_movement_crouch_friction
	OC_const_movement_stand_friction_threshold
	OC_const_movement_crouch_friction_threshold
	OC_const_movement_air_gethit_groundlevel
	OC_const_movement_air_gethit_groundrecover_ground_threshold
	OC_const_movement_air_gethit_groundrecover_groundlevel
	OC_const_movement_air_gethit_airrecover_threshold
	OC_const_movement_air_gethit_airrecover_yaccel
	OC_const_movement_air_gethit_trip_groundlevel
	OC_const_movement_down_bounce_offset_x
	OC_const_movement_down_bounce_offset_y
	OC_const_movement_down_bounce_yaccel
	OC_const_movement_down_bounce_groundlevel
	OC_const_movement_down_gethit_offset_x
	OC_const_movement_down_gethit_offset_y
	OC_const_movement_down_friction_threshold
	OC_const_name
	OC_const_p2name
	OC_const_p3name
	OC_const_p4name
	OC_const_p5name
	OC_const_p6name
	OC_const_p7name
	OC_const_p8name
	OC_const_authorname
	OC_const_displayname
	OC_const_stagevar_info_author
	OC_const_stagevar_info_displayname
	OC_const_stagevar_info_name
	OC_const_stagevar_camera_boundleft
	OC_const_stagevar_camera_boundright
	OC_const_stagevar_camera_boundhigh
	OC_const_stagevar_camera_boundlow
	OC_const_stagevar_camera_verticalfollow
	OC_const_stagevar_camera_floortension
	OC_const_stagevar_camera_tensionhigh
	OC_const_stagevar_camera_tensionlow
	OC_const_stagevar_camera_tension
	OC_const_stagevar_camera_tensionvel
	OC_const_stagevar_camera_cuthigh
	OC_const_stagevar_camera_cutlow
	OC_const_stagevar_camera_startzoom
	OC_const_stagevar_camera_zoomout
	OC_const_stagevar_camera_zoomin
	OC_const_stagevar_camera_zoomindelay
	OC_const_stagevar_camera_zoominspeed
	OC_const_stagevar_camera_zoomoutspeed
	OC_const_stagevar_camera_yscrollspeed
	OC_const_stagevar_camera_ytension_enable
	OC_const_stagevar_camera_autocenter
	OC_const_stagevar_camera_lowestcap
	OC_const_stagevar_playerinfo_leftbound
	OC_const_stagevar_playerinfo_rightbound
	OC_const_stagevar_playerinfo_topbound
	OC_const_stagevar_playerinfo_botbound
	OC_const_stagevar_scaling_topz
	OC_const_stagevar_scaling_botz
	OC_const_stagevar_scaling_topscale
	OC_const_stagevar_scaling_botscale
	OC_const_stagevar_bound_screenleft
	OC_const_stagevar_bound_screenright
	OC_const_stagevar_stageinfo_localcoord_x
	OC_const_stagevar_stageinfo_localcoord_y
	OC_const_stagevar_stageinfo_xscale
	OC_const_stagevar_stageinfo_yscale
	OC_const_stagevar_stageinfo_zoffset
	OC_const_stagevar_stageinfo_zoffsetlink
	OC_const_stagevar_shadow_intensity
	OC_const_stagevar_shadow_color_r
	OC_const_stagevar_shadow_color_g
	OC_const_stagevar_shadow_color_b
	OC_const_stagevar_shadow_yscale
	OC_const_stagevar_shadow_fade_range_begin
	OC_const_stagevar_shadow_fade_range_end
	OC_const_stagevar_shadow_xshear
	OC_const_stagevar_shadow_offset_x
	OC_const_stagevar_shadow_offset_y
	OC_const_stagevar_reflection_intensity
	OC_const_stagevar_reflection_yscale
	OC_const_stagevar_reflection_offset_x
	OC_const_stagevar_reflection_offset_y
	OC_const_stagevar_reflection_xshear
	OC_const_stagevar_reflection_color_r
	OC_const_stagevar_reflection_color_g
	OC_const_stagevar_reflection_color_b
	OC_const_gameoption
	OC_const_constants
	OC_const_stage_constants
)
const (
	OC_st_var OpCode = iota
	OC_st_sysvar
	OC_st_fvar
	OC_st_sysfvar
	OC_st_varadd
	OC_st_sysvaradd
	OC_st_fvaradd
	OC_st_sysfvaradd
	OC_st_map
)
const (
	OC_ex_p2dist_x OpCode = iota
	OC_ex_p2dist_y
	OC_ex_p2dist_z
	OC_ex_p2bodydist_x
	OC_ex_p2bodydist_y
	OC_ex_p2bodydist_z
	OC_ex_parentdist_x
	OC_ex_parentdist_y
	OC_ex_parentdist_z
	OC_ex_rootdist_x
	OC_ex_rootdist_y
	OC_ex_rootdist_z
	OC_ex_win
	OC_ex_winko
	OC_ex_wintime
	OC_ex_winperfect
	OC_ex_winspecial
	OC_ex_winhyper
	OC_ex_lose
	OC_ex_loseko
	OC_ex_losetime
	OC_ex_drawgame
	OC_ex_matchover
	OC_ex_matchno
	OC_ex_roundno
	OC_ex_roundsexisted
	OC_ex_ishometeam
	OC_ex_tickspersecond
	OC_ex_const240p
	OC_ex_const480p
	OC_ex_const720p
	OC_ex_const1080p
	OC_ex_gethitvar_animtype
	OC_ex_gethitvar_air_animtype
	OC_ex_gethitvar_ground_animtype
	OC_ex_gethitvar_fall_animtype
	OC_ex_gethitvar_type
	OC_ex_gethitvar_airtype
	OC_ex_gethitvar_groundtype
	OC_ex_gethitvar_damage
	OC_ex_gethitvar_hitcount
	OC_ex_gethitvar_fallcount
	OC_ex_gethitvar_hitshaketime
	OC_ex_gethitvar_hittime
	OC_ex_gethitvar_slidetime
	OC_ex_gethitvar_ctrltime
	OC_ex_gethitvar_xoff
	OC_ex_gethitvar_yoff
	OC_ex_gethitvar_zoff
	OC_ex_gethitvar_xvel
	OC_ex_gethitvar_yvel
	OC_ex_gethitvar_zvel
	OC_ex_gethitvar_xaccel
	OC_ex_gethitvar_yaccel
	OC_ex_gethitvar_zaccel
	OC_ex_gethitvar_xveladd
	OC_ex_gethitvar_yveladd
	OC_ex_gethitvar_chainid
	OC_ex_gethitvar_guarded
	OC_ex_gethitvar_isbound
	OC_ex_gethitvar_fall
	OC_ex_gethitvar_fall_damage
	OC_ex_gethitvar_fall_xvel
	OC_ex_gethitvar_fall_yvel
	OC_ex_gethitvar_fall_zvel
	OC_ex_gethitvar_fall_recover
	OC_ex_gethitvar_fall_time
	OC_ex_gethitvar_fall_recovertime
	OC_ex_gethitvar_fall_kill
	OC_ex_gethitvar_fall_envshake_time
	OC_ex_gethitvar_fall_envshake_freq
	OC_ex_gethitvar_fall_envshake_ampl
	OC_ex_gethitvar_fall_envshake_phase
	OC_ex_gethitvar_fall_envshake_mul
	OC_ex_gethitvar_attr
	OC_ex_gethitvar_dizzypoints
	OC_ex_gethitvar_guardpoints
	OC_ex_gethitvar_id
	OC_ex_gethitvar_playerno
	OC_ex_gethitvar_redlife
	OC_ex_gethitvar_score
	OC_ex_gethitvar_hitdamage
	OC_ex_gethitvar_guarddamage
	OC_ex_gethitvar_power
	OC_ex_gethitvar_hitpower
	OC_ex_gethitvar_guardpower
	OC_ex_gethitvar_kill
	OC_ex_gethitvar_priority
	OC_ex_gethitvar_guardcount
	OC_ex_gethitvar_facing
	OC_ex_gethitvar_ground_velocity_x
	OC_ex_gethitvar_ground_velocity_y
	OC_ex_gethitvar_ground_velocity_z
	OC_ex_gethitvar_air_velocity_x
	OC_ex_gethitvar_air_velocity_y
	OC_ex_gethitvar_air_velocity_z
	OC_ex_gethitvar_down_velocity_x
	OC_ex_gethitvar_down_velocity_y
	OC_ex_gethitvar_down_velocity_z
	OC_ex_gethitvar_guard_velocity_x
	OC_ex_gethitvar_guard_velocity_y
	OC_ex_gethitvar_guard_velocity_z
	OC_ex_gethitvar_airguard_velocity_x
	OC_ex_gethitvar_airguard_velocity_y
	OC_ex_gethitvar_airguard_velocity_z
	OC_ex_gethitvar_frame
	OC_ex_gethitvar_down_recover
	OC_ex_gethitvar_down_recovertime
	OC_ex_gethitvar_guardflag
	OC_ex_ailevelf
	OC_ex_animelemvar_alphadest
	OC_ex_animelemvar_angle
	OC_ex_animelemvar_alphasource
	OC_ex_animelemvar_group
	OC_ex_animelemvar_hflip
	OC_ex_animelemvar_image
	OC_ex_animelemvar_time
	OC_ex_animelemvar_vflip
	OC_ex_animelemvar_xoffset
	OC_ex_animelemvar_xscale
	OC_ex_animelemvar_yoffset
	OC_ex_animelemvar_yscale
	OC_ex_animelemvar_numclsn1
	OC_ex_animelemvar_numclsn2
	OC_ex_animlength
	OC_ex_animplayerno
	OC_ex_attack
	OC_ex_clsnoverlap
	OC_ex_combocount
	OC_ex_consecutivewins
	OC_ex_decisiveround
	OC_ex_defence
	OC_ex_dizzy
	OC_ex_dizzypoints
	OC_ex_dizzypointsmax
	OC_ex_fighttime
	OC_ex_firstattack
	OC_ex_float
	OC_ex_gamemode
	OC_ex_groundangle
	OC_ex_guardbreak
	OC_ex_guardpoints
	OC_ex_guardpointsmax
	OC_ex_helperid
	OC_ex_helperindexexist
	OC_ex_helpername
	OC_ex_hitoverridden
	OC_ex_inputtime_B
	OC_ex_inputtime_D
	OC_ex_inputtime_F
	OC_ex_inputtime_U
	OC_ex_inputtime_L
	OC_ex_inputtime_R
	OC_ex_inputtime_a
	OC_ex_inputtime_b
	OC_ex_inputtime_c
	OC_ex_inputtime_x
	OC_ex_inputtime_y
	OC_ex_inputtime_z
	OC_ex_inputtime_s
	OC_ex_inputtime_d
	OC_ex_inputtime_w
	OC_ex_inputtime_m
	OC_ex_movehitvar_frame
	OC_ex_movehitvar_cornerpush
	OC_ex_movehitvar_id
	OC_ex_movehitvar_overridden
	OC_ex_movehitvar_playerno
	OC_ex_movehitvar_spark_x
	OC_ex_movehitvar_spark_y
	OC_ex_movehitvar_uniqhit
	OC_ex_ikemenversion
	OC_ex_incustomanim
	OC_ex_incustomstate
	OC_ex_indialogue
	OC_ex_isassertedchar
	OC_ex_isassertedglobal
	OC_ex_ishost
	OC_ex_jugglepoints
	OC_ex_localcoord_x
	OC_ex_localcoord_y
	OC_ex_maparray
	OC_ex_max
	OC_ex_min
	OC_ex_numplayer
	OC_ex_clamp
	OC_ex_sign
	OC_ex_atan2
	OC_ex_rad
	OC_ex_deg
	OC_ex_lastplayerid
	OC_ex_lerp
	OC_ex_memberno
	OC_ex_movecountered
	OC_ex_mugenversion
	OC_ex_pausetime
	OC_ex_physics
	OC_ex_playerno
	OC_ex_playerindexexist
	OC_ex_playernoexist
	OC_ex_randomrange
	OC_ex_ratiolevel
	OC_ex_receiveddamage
	OC_ex_receivedhits
	OC_ex_redlife
	OC_ex_round
	OC_ex_roundtime
	OC_ex_score
	OC_ex_scoretotal
	OC_ex_selfstatenoexist
	OC_ex_sprpriority
	OC_ex_stagebackedgedist
	OC_ex_stagefrontedgedist
	OC_ex_stagetime
	OC_ex_standby
	OC_ex_teamleader
	OC_ex_teamsize
	OC_ex_timeelapsed
	OC_ex_timeremaining
	OC_ex_timetotal
	OC_ex_playercount
	OC_ex_pos_z
	OC_ex_vel_z
	OC_ex_prevanim
	OC_ex_prevmovetype
	OC_ex_prevstatetype
	OC_ex_reversaldefattr
	OC_ex_airjumpcount
	OC_ex_envshakevar_time
	OC_ex_envshakevar_freq
	OC_ex_envshakevar_ampl
	OC_ex_angle
	OC_ex_scale_x
	OC_ex_scale_y
	OC_ex_scale_z
	OC_ex_offset_x
	OC_ex_offset_y
	OC_ex_alpha_s
	OC_ex_alpha_d
	OC_ex_selfcommand
	OC_ex_guardcount
	OC_ex_fightscreenvar_info_author
	OC_ex_fightscreenvar_info_localcoord_x
	OC_ex_fightscreenvar_info_localcoord_y
	OC_ex_fightscreenvar_info_name
	OC_ex_fightscreenvar_round_ctrl_time
	OC_ex_fightscreenvar_round_over_hittime
	OC_ex_fightscreenvar_round_over_time
	OC_ex_fightscreenvar_round_over_waittime
	OC_ex_fightscreenvar_round_over_wintime
	OC_ex_fightscreenvar_round_slow_time
	OC_ex_fightscreenvar_round_start_waittime
	OC_ex_fightscreenvar_round_callfight_time
	OC_ex_fightscreenvar_time_framespercount
	OC_ex_groundlevel
	OC_ex_layerno
)
const (
	OC_ex2_index OpCode = iota
	OC_ex2_runorder
	OC_ex2_palfxvar_time
	OC_ex2_palfxvar_addr
	OC_ex2_palfxvar_addg
	OC_ex2_palfxvar_addb
	OC_ex2_palfxvar_mulr
	OC_ex2_palfxvar_mulg
	OC_ex2_palfxvar_mulb
	OC_ex2_palfxvar_color
	OC_ex2_palfxvar_hue
	OC_ex2_palfxvar_invertall
	OC_ex2_palfxvar_invertblend
	OC_ex2_palfxvar_bg_time
	OC_ex2_palfxvar_bg_addr
	OC_ex2_palfxvar_bg_addg
	OC_ex2_palfxvar_bg_addb
	OC_ex2_palfxvar_bg_mulr
	OC_ex2_palfxvar_bg_mulg
	OC_ex2_palfxvar_bg_mulb
	OC_ex2_palfxvar_bg_color
	OC_ex2_palfxvar_bg_hue
	OC_ex2_palfxvar_bg_invertall
	OC_ex2_palfxvar_all_time
	OC_ex2_palfxvar_all_addr
	OC_ex2_palfxvar_all_addg
	OC_ex2_palfxvar_all_addb
	OC_ex2_palfxvar_all_mulr
	OC_ex2_palfxvar_all_mulg
	OC_ex2_palfxvar_all_mulb
	OC_ex2_palfxvar_all_color
	OC_ex2_palfxvar_all_hue
	OC_ex2_palfxvar_all_invertall
	OC_ex2_palfxvar_all_invertblend
	OC_ex2_introstate
	OC_ex2_outrostate
	OC_ex2_bgmvar_filename
	OC_ex2_bgmvar_freqmul
	OC_ex2_bgmvar_length
	OC_ex2_bgmvar_loop
	OC_ex2_bgmvar_loopcount
	OC_ex2_bgmvar_loopend
	OC_ex2_bgmvar_loopstart
	OC_ex2_bgmvar_position
	OC_ex2_bgmvar_startposition
	OC_ex2_bgmvar_volume
	OC_ex2_clsnvar_left
	OC_ex2_clsnvar_top
	OC_ex2_clsnvar_right
	OC_ex2_clsnvar_bottom
	OC_ex2_debug_accel
	OC_ex2_debug_clsndisplay
	OC_ex2_debug_debugdisplay
	OC_ex2_debug_lifebardisplay
	OC_ex2_debug_roundreset
	OC_ex2_debug_wireframedisplay
	OC_ex2_drawpal_group
	OC_ex2_drawpal_index
	OC_ex2_explodvar_accel_x
	OC_ex2_explodvar_accel_y
	OC_ex2_explodvar_accel_z
	OC_ex2_explodvar_angle
	OC_ex2_explodvar_angle_x
	OC_ex2_explodvar_angle_y
	OC_ex2_explodvar_anim
	OC_ex2_explodvar_animelem
	OC_ex2_explodvar_bindtime
	OC_ex2_explodvar_drawpal_group
	OC_ex2_explodvar_drawpal_index
	OC_ex2_explodvar_facing
	OC_ex2_explodvar_friction_x
	OC_ex2_explodvar_friction_y
	OC_ex2_explodvar_friction_z
	OC_ex2_explodvar_id
	OC_ex2_explodvar_layerno
	OC_ex2_explodvar_pausemovetime
	OC_ex2_explodvar_pos_x
	OC_ex2_explodvar_pos_y
	OC_ex2_explodvar_pos_z
	OC_ex2_explodvar_removetime
	OC_ex2_explodvar_scale_x
	OC_ex2_explodvar_scale_y
	OC_ex2_explodvar_sprpriority
	OC_ex2_explodvar_time
	OC_ex2_explodvar_vel_x
	OC_ex2_explodvar_vel_y
	OC_ex2_explodvar_vel_z
	OC_ex2_explodvar_xshear
	OC_ex2_projvar_accel_x
	OC_ex2_projvar_accel_y
	OC_ex2_projvar_accel_z
	OC_ex2_projvar_animelem
	OC_ex2_projvar_drawpal_group
	OC_ex2_projvar_drawpal_index
	OC_ex2_projvar_facing
	OC_ex2_projvar_guardflag
	OC_ex2_projvar_highbound
	OC_ex2_projvar_hitflag
	OC_ex2_projvar_lowbound
	OC_ex2_projvar_pausemovetime
	OC_ex2_projvar_pos_x
	OC_ex2_projvar_pos_y
	OC_ex2_projvar_pos_z
	OC_ex2_projvar_projangle
	OC_ex2_projvar_projanim
	OC_ex2_projvar_projcancelanim
	OC_ex2_projvar_projedgebound
	OC_ex2_projvar_projhitanim
	OC_ex2_projvar_projhits
	OC_ex2_projvar_projhitsmax
	OC_ex2_projvar_projid
	OC_ex2_projvar_projlayerno
	OC_ex2_projvar_projmisstime
	OC_ex2_projvar_projpriority
	OC_ex2_projvar_projremanim
	OC_ex2_projvar_projremove
	OC_ex2_projvar_projremovetime
	OC_ex2_projvar_projscale_x
	OC_ex2_projvar_projscale_y
	OC_ex2_projvar_projshadow_b
	OC_ex2_projvar_projshadow_g
	OC_ex2_projvar_projshadow_r
	OC_ex2_projvar_projsprpriority
	OC_ex2_projvar_projstagebound
	OC_ex2_projvar_projxshear
	OC_ex2_projvar_remvelocity_x
	OC_ex2_projvar_remvelocity_y
	OC_ex2_projvar_remvelocity_z
	OC_ex2_projvar_supermovetime
	OC_ex2_projvar_teamside
	OC_ex2_projvar_time
	OC_ex2_projvar_vel_x
	OC_ex2_projvar_vel_y
	OC_ex2_projvar_vel_z
	OC_ex2_projvar_velmul_x
	OC_ex2_projvar_velmul_y
	OC_ex2_projvar_velmul_z
	OC_ex2_hitdefvar_guard_pausetime
	OC_ex2_hitdefvar_guard_shaketime
	OC_ex2_hitdefvar_guard_sparkno
	OC_ex2_hitdefvar_guarddamage
	OC_ex2_hitdefvar_guardflag
	OC_ex2_hitdefvar_guardsound_group
	OC_ex2_hitdefvar_guardsound_number
	OC_ex2_hitdefvar_hitdamage
	OC_ex2_hitdefvar_hitflag
	OC_ex2_hitdefvar_hitsound_group
	OC_ex2_hitdefvar_hitsound_number
	OC_ex2_hitdefvar_id
	OC_ex2_hitdefvar_p1stateno
	OC_ex2_hitdefvar_p2stateno
	OC_ex2_hitdefvar_pausetime
	OC_ex2_hitdefvar_priority
	OC_ex2_hitdefvar_shaketime
	OC_ex2_hitdefvar_sparkno
	OC_ex2_hitdefvar_sparkx
	OC_ex2_hitdefvar_sparky
	OC_ex2_hitbyattr
	OC_ex2_soundvar_group
	OC_ex2_soundvar_number
	OC_ex2_soundvar_freqmul
	OC_ex2_soundvar_isplaying
	OC_ex2_soundvar_length
	OC_ex2_soundvar_loopcount
	OC_ex2_soundvar_loopstart
	OC_ex2_soundvar_loopend
	OC_ex2_soundvar_pan
	OC_ex2_soundvar_position
	OC_ex2_soundvar_priority
	OC_ex2_soundvar_startposition
	OC_ex2_soundvar_volumescale
	OC_ex2_fightscreenstate_fightdisplay
	OC_ex2_fightscreenstate_kodisplay
	OC_ex2_fightscreenstate_rounddisplay
	OC_ex2_fightscreenstate_windisplay
	OC_ex2_motifstate_continuescreen
	OC_ex2_motifstate_victoryscreen
	OC_ex2_motifstate_winscreen
	OC_ex2_systemvar_introtime
	OC_ex2_systemvar_outrotime
	OC_ex2_systemvar_pausetime
	OC_ex2_systemvar_slowtime
	OC_ex2_systemvar_superpausetime
	OC_ex2_topbounddist
	OC_ex2_topboundbodydist
	OC_ex2_botbounddist
	OC_ex2_botboundbodydist
	OC_ex2_stagebgvar_actionno
	OC_ex2_stagebgvar_delta_x
	OC_ex2_stagebgvar_delta_y
	OC_ex2_stagebgvar_id
	OC_ex2_stagebgvar_layerno
	OC_ex2_stagebgvar_pos_x
	OC_ex2_stagebgvar_pos_y
	OC_ex2_stagebgvar_start_x
	OC_ex2_stagebgvar_start_y
	OC_ex2_stagebgvar_tile_x
	OC_ex2_stagebgvar_tile_y
	OC_ex2_stagebgvar_velocity_x
	OC_ex2_stagebgvar_velocity_y
	OC_ex2_numstagebg
	OC_ex2_envshakevar_dir
	OC_ex2_gethitvar_fall_envshake_dir
)

type StringPool struct {
	List []string
	Map  map[string]int
}

func NewStringPool() *StringPool {
	return &StringPool{Map: make(map[string]int)}
}

func (sp *StringPool) Clear() {
	sp.List, sp.Map = nil, make(map[string]int)
}

func (sp *StringPool) Add(s string) int {
	i, ok := sp.Map[s]
	if !ok {
		i = len(sp.List)
		sp.List = append(sp.List, s)
		sp.Map[s] = i
	}
	return i
}

type BytecodeValue struct {
	vtype ValueType
	value float64
}

func (bv BytecodeValue) IsNone() bool {
	return bv.vtype == VT_None
}

func (bv BytecodeValue) IsSF() bool {
	return bv.vtype == VT_SFalse
}

func (bv BytecodeValue) ToF() float32 {
	if bv.IsSF() {
		return 0
	}
	return float32(bv.value)
}

func (bv BytecodeValue) ToI() int32 {
	if bv.IsSF() {
		return 0
	}
	return int32(bv.value)
}

func (bv BytecodeValue) ToI64() int64 {
	if bv.IsSF() {
		return 0
	}
	return int64(bv.value)
}

func (bv BytecodeValue) ToB() bool {
	if bv.IsSF() || bv.value == 0 {
		return false
	}
	return true
}

func (bv *BytecodeValue) SetF(f float32) {
	if math.IsNaN(float64(f)) {
		*bv = BytecodeSF()
	} else {
		*bv = BytecodeValue{VT_Float, float64(f)}
	}
}

func (bv *BytecodeValue) SetI(i int32) {
	*bv = BytecodeValue{VT_Int, float64(i)}
}

func (bv *BytecodeValue) SetI64(i int64) {
	*bv = BytecodeValue{VT_Int, float64(i)}
}

func (bv *BytecodeValue) SetB(b bool) {
	bv.vtype = VT_Bool
	bv.value = float64(Btoi(b))
}

func bvNone() BytecodeValue {
	return BytecodeValue{VT_None, 0}
}

func BytecodeSF() BytecodeValue {
	return BytecodeValue{VT_SFalse, math.NaN()}
}

func BytecodeFloat(f float32) BytecodeValue {
	return BytecodeValue{VT_Float, float64(f)}
}

func BytecodeInt(i int32) BytecodeValue {
	return BytecodeValue{VT_Int, float64(i)}
}

func BytecodeInt64(i int64) BytecodeValue {
	return BytecodeValue{VT_Int, float64(i)}
}

func BytecodeBool(b bool) BytecodeValue {
	return BytecodeValue{VT_Bool, float64(Btoi(b))}
}

type BytecodeStack []BytecodeValue

func (bs *BytecodeStack) Clear() {
	*bs = (*bs)[:0]
}

func (bs *BytecodeStack) Push(bv BytecodeValue) {
	*bs = append(*bs, bv)
}

func (bs *BytecodeStack) PushI(i int32) {
	bs.Push(BytecodeInt(i))
}

func (bs *BytecodeStack) PushI64(i int64) {
	bs.Push(BytecodeInt64(i))
}

func (bs *BytecodeStack) PushF(f float32) {
	bs.Push(BytecodeFloat(f))
}

func (bs *BytecodeStack) PushB(b bool) {
	bs.Push(BytecodeBool(b))
}

func (bs BytecodeStack) Top() *BytecodeValue {
	// This should only happen during development
	if len(bs) == 0 {
		panic(Error("Attempted to access the top of an empty ByteCode stack.\n"))
	}

	return &bs[len(bs)-1]
}

func (bs *BytecodeStack) Pop() (bv BytecodeValue) {
	// This should only happen during development
	if len(*bs) == 0 {
		panic(Error("Attempted to pop from an empty ByteCode stack.\n"))
	}

	// Set value to what's at the top of stack. Shift stack
	bv, *bs = *bs.Top(), (*bs)[:len(*bs)-1]

	return
}

func (bs *BytecodeStack) Dup() {
	bs.Push(*bs.Top())
}

func (bs *BytecodeStack) Swap() {
	*bs.Top(), (*bs)[len(*bs)-2] = (*bs)[len(*bs)-2], *bs.Top()
}

func (bs *BytecodeStack) Alloc(size int) []BytecodeValue {
	if len(*bs)+size > cap(*bs) {
		tmp := *bs
		*bs = make(BytecodeStack, len(*bs)+size)
		copy(*bs, tmp)
	} else {
		*bs = (*bs)[:len(*bs)+size]
		for i := len(*bs) - size; i < len(*bs); i++ {
			(*bs)[i] = bvNone()
		}
	}
	return (*bs)[len(*bs)-size:]
}

type BytecodeExp []OpCode

func Float32frombytes(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func (be *BytecodeExp) append(op ...OpCode) {
	*be = append(*be, op...)
}

func (be *BytecodeExp) appendValue(bv BytecodeValue) (ok bool) {
	switch bv.vtype {
	case VT_Float:
		be.append(OC_float)
		f := float32(bv.value)
		be.append((*(*[4]OpCode)(unsafe.Pointer(&f)))[:]...)
	case VT_Int:
		if bv.value >= -128 && bv.value <= 127 {
			be.append(OC_int8, OpCode(bv.value))
		} else if bv.value >= math.MinInt32 && bv.value <= math.MaxInt32 {
			be.append(OC_int)
			i := int32(bv.value)
			be.append((*(*[4]OpCode)(unsafe.Pointer(&i)))[:]...)
		} else {
			be.append(OC_int64)
			i := int64(bv.value)
			be.append((*(*[8]OpCode)(unsafe.Pointer(&i)))[:]...)
		}
	case VT_Bool:
		if bv.value != 0 {
			be.append(OC_int8, 1)
		} else {
			be.append(OC_int8, 0)
		}
	case VT_SFalse:
		be.append(OC_int8, 0)
	default:
		return false
	}
	return true
}

// Appends multiple int32 operands to the BytecodeExp
func (be *BytecodeExp) appendI32s(addrs ...int32) {
	for _, addr := range addrs {
		be.append((*(*[4]OpCode)(unsafe.Pointer(&addr)))[:]...)
	}
}

// Pushes an OpCode with an int32 operand to the top of the BytecodeExp.
func (be *BytecodeExp) appendI32Op(op OpCode, addr int32) {
	be.append(op)
	be.append((*(*[4]OpCode)(unsafe.Pointer(&addr)))[:]...)
}

// Pushes an OpCode with an int64 operand to the top of the BytecodeExp.
func (be *BytecodeExp) appendI64Op(op OpCode, addr int64) {
	be.append(op)
	be.append((*(*[8]OpCode)(unsafe.Pointer(&addr)))[:]...)
}

func (BytecodeExp) neg(v *BytecodeValue) {
	if v.vtype == VT_Float {
		v.value *= -1
	} else {
		v.SetI(-v.ToI())
	}
}

func (BytecodeExp) not(v *BytecodeValue) {
	v.SetI(^v.ToI())
}

func (BytecodeExp) blnot(v *BytecodeValue) {
	v.SetB(!v.ToB())
}

func (BytecodeExp) pow(v1 *BytecodeValue, v2 BytecodeValue, pn int) {
	// This one's interesting in Mugen because 0**-1 is not treated the same as 1/0
	// In Mugen 1.1 it's considered infinity
	// In WinMugen it's considered infinity if it's called as a float, but result alternates between 0 and 2**31 if called as an int
	// These bugs are not reproduced in Ikemen
	// TODO: Perhaps Ikemen characters should treat 0**-1 the same as 1/0

	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float || v2.ToF() < 0 {
		// Float power
		v1.SetF(Pow(v1.ToF(), v2.ToF()))
	} else {
		// Int power
		i1, i2, hb := v1.ToI(), v2.ToI(), int32(-1)
		for uint32(i2)>>uint(hb+1) != 0 {
			hb++
		}
		var i, bit, tmp int32 = 1, 0, i1
		for ; bit <= hb; bit++ {
			var shift uint
			if bit == hb || sys.cgi[pn].mugenver[0] == 1 {
				shift = uint(bit)
			} else {
				shift = uint((hb - 1) - bit)
			}
			if i2&(1<<shift) != 0 {
				i *= tmp
			}
			tmp *= tmp
		}
		v1.SetI(i)
	}

	// Print error for invalid operations
	result := float64(v1.ToF())
	if math.IsNaN(result) || math.IsInf(result, 0) {
		sys.printBytecodeError("Invalid exponentiation")
	}
}

func (BytecodeExp) mul(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetF(v1.ToF() * v2.ToF())
	} else {
		v1.SetI(v1.ToI() * v2.ToI())
	}
}

func (BytecodeExp) div(v1 *BytecodeValue, v2 BytecodeValue) {
	if v2.ToF() == 0 {
		// Division by 0
		*v1 = BytecodeSF()
		sys.printBytecodeError("Division by 0")
	} else if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		// Float division
		v1.SetF(v1.ToF() / v2.ToF())
	} else {
		// Int division
		v1.SetI(v1.ToI() / v2.ToI())
	}
}

func (BytecodeExp) mod(v1 *BytecodeValue, v2 BytecodeValue) {
	if v2.ToI() == 0 {
		*v1 = BytecodeSF()
		sys.printBytecodeError("Modulus by 0")
	} else {
		v1.SetI(v1.ToI() % v2.ToI())
	}
}

func (BytecodeExp) add(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetF(v1.ToF() + v2.ToF())
	} else {
		v1.SetI(v1.ToI() + v2.ToI())
	}
}

func (BytecodeExp) sub(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetF(v1.ToF() - v2.ToF())
	} else {
		v1.SetI(v1.ToI() - v2.ToI())
	}
}

func (BytecodeExp) gt(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetB(v1.ToF() > v2.ToF())
	} else {
		v1.SetB(v1.ToI() > v2.ToI())
	}
}

func (BytecodeExp) ge(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetB(v1.ToF() >= v2.ToF())
	} else {
		v1.SetB(v1.ToI() >= v2.ToI())
	}
}

func (BytecodeExp) lt(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetB(v1.ToF() < v2.ToF())
	} else {
		v1.SetB(v1.ToI() < v2.ToI())
	}
}

func (BytecodeExp) le(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetB(v1.ToF() <= v2.ToF())
	} else {
		v1.SetB(v1.ToI() <= v2.ToI())
	}
}

func (BytecodeExp) eq(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetB(v1.ToF() == v2.ToF())
	} else {
		v1.SetB(v1.ToI() == v2.ToI())
	}
}

func (BytecodeExp) ne(v1 *BytecodeValue, v2 BytecodeValue) {
	if ValueType(Min(int32(v1.vtype), int32(v2.vtype))) == VT_Float {
		v1.SetB(v1.ToF() != v2.ToF())
	} else {
		v1.SetB(v1.ToI() != v2.ToI())
	}
}

func (BytecodeExp) and(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetI(v1.ToI() & v2.ToI())
}

func (BytecodeExp) xor(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetI(v1.ToI() ^ v2.ToI())
}

func (BytecodeExp) or(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetI(v1.ToI() | v2.ToI())
}

func (BytecodeExp) bland(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetB(v1.ToB() && v2.ToB())
}

func (BytecodeExp) blxor(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetB(v1.ToB() != v2.ToB())
}

func (BytecodeExp) blor(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetB(v1.ToB() || v2.ToB())
}

func (BytecodeExp) abs(v1 *BytecodeValue) {
	if v1.vtype == VT_Float {
		v1.value = math.Abs(v1.value)
	} else {
		v1.SetI(Abs(v1.ToI()))
	}
}

func (BytecodeExp) exp(v1 *BytecodeValue) {
	v1.SetF(float32(math.Exp(v1.value)))
}

func (BytecodeExp) ln(v1 *BytecodeValue) {
	if v1.value <= 0 {
		*v1 = BytecodeSF()
		sys.printBytecodeError("Invalid logarithm")
	} else {
		v1.SetF(float32(math.Log(v1.value)))
	}
}

func (BytecodeExp) log(v1 *BytecodeValue, v2 BytecodeValue) {
	if v1.value <= 0 || v2.value <= 0 {
		*v1 = BytecodeSF()
		sys.printBytecodeError("Invalid logarithm")
	} else {
		v1.SetF(float32(math.Log(v2.value) / math.Log(v1.value)))
	}
}

func (BytecodeExp) cos(v1 *BytecodeValue) {
	v1.SetF(float32(math.Cos(v1.value)))
}

func (BytecodeExp) sin(v1 *BytecodeValue) {
	v1.SetF(float32(math.Sin(v1.value)))
}

func (BytecodeExp) tan(v1 *BytecodeValue) {
	v1.SetF(float32(math.Tan(v1.value)))
}

func (BytecodeExp) acos(v1 *BytecodeValue) {
	v1.SetF(float32(math.Acos(v1.value)))
}

func (BytecodeExp) asin(v1 *BytecodeValue) {
	v1.SetF(float32(math.Asin(v1.value)))
}

func (BytecodeExp) atan(v1 *BytecodeValue) {
	v1.SetF(float32(math.Atan(v1.value)))
}

func (BytecodeExp) floor(v1 *BytecodeValue) {
	if v1.vtype == VT_Float {
		f := math.Floor(v1.value)
		if math.IsNaN(f) {
			*v1 = BytecodeSF()
		} else {
			v1.SetI(int32(f))
		}
	}
}

func (BytecodeExp) ceil(v1 *BytecodeValue) {
	if v1.vtype == VT_Float {
		f := math.Ceil(v1.value)
		if math.IsNaN(f) {
			*v1 = BytecodeSF()
		} else {
			v1.SetI(int32(f))
		}
	}
}

func (BytecodeExp) max(v1 *BytecodeValue, v2 BytecodeValue) {
	if v1.value >= v2.value {
		v1.SetF(float32(v1.value))
	} else {
		v1.SetF(float32(v2.value))
	}
}

func (BytecodeExp) min(v1 *BytecodeValue, v2 BytecodeValue) {
	if v1.value <= v2.value {
		v1.SetF(float32(v1.value))
	} else {
		v1.SetF(float32(v2.value))
	}
}

func (BytecodeExp) random(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetI(RandI(int32(v1.value), int32(v2.value)))
}

func (BytecodeExp) round(v1 *BytecodeValue, v2 BytecodeValue) {
	shift := math.Pow(10, v2.value)
	v1.SetF(float32(math.Floor((v1.value*shift)+0.5) / shift))
}

func (BytecodeExp) clamp(v1 *BytecodeValue, v2 BytecodeValue, v3 BytecodeValue) {
	if v1.value <= v2.value {
		v1.SetF(float32(v2.value))
	} else if v1.value >= v3.value {
		v1.SetF(float32(v3.value))
	} else {
		v1.SetF(float32(v1.value))
	}
}

func (BytecodeExp) atan2(v1 *BytecodeValue, v2 BytecodeValue) {
	v1.SetF(float32(math.Atan2(v1.value, v2.value)))
}

func (BytecodeExp) sign(v1 *BytecodeValue) {
	if v1.value < 0 {
		v1.SetI(int32(-1))
	} else if v1.value > 0 {
		v1.SetI(int32(1))
	} else {
		v1.SetI(int32(0))
	}
}

func (BytecodeExp) rad(v1 *BytecodeValue) {
	v1.SetF(float32(v1.value * math.Pi / 180))
}

func (BytecodeExp) deg(v1 *BytecodeValue) {
	v1.SetF(float32(v1.value * 180 / math.Pi))
}

func (BytecodeExp) lerp(v1 *BytecodeValue, v2 BytecodeValue, v3 BytecodeValue) {
	amount := v3.value
	if v3.value <= 0 {
		amount = 0
	} else if v3.value >= 1 {
		amount = 1
	}
	v1.SetF(float32(v1.value + (v2.value-v1.value)*amount))
}

func (be BytecodeExp) run(c *Char) BytecodeValue {
	oc := c
	for i := 1; i <= len(be); i++ {
		switch be[i-1] {
		case OC_jsf8:
			if sys.bcStack.Top().IsSF() {
				if be[i] == 0 {
					i = len(be)
				} else {
					i += int(uint8(be[i])) + 1
				}
			} else {
				i++
			}
		case OC_jz8, OC_jnz8:
			if sys.bcStack.Top().ToB() == (be[i-1] == OC_jz8) {
				i++
				break
			}
			fallthrough
		case OC_jmp8:
			if be[i] == 0 {
				i = len(be)
			} else {
				i += int(uint8(be[i])) + 1
			}
		case OC_jz, OC_jnz:
			if sys.bcStack.Top().ToB() == (be[i-1] == OC_jz) {
				i += 4
				break
			}
			fallthrough
		case OC_jmp:
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_player:
			if c = sys.playerID(c.getPlayerID(int(sys.bcStack.Pop().ToI()))); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_parent:
			if c = c.parent(true); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_root:
			if c = c.root(true); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_helper:
			v2 := sys.bcStack.Pop().ToI()
			v1 := sys.bcStack.Pop().ToI()
			if c = c.helperTrigger(v1, int(v2)); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_target:
			v2 := sys.bcStack.Pop().ToI()
			v1 := sys.bcStack.Pop().ToI()
			if c = c.targetTrigger(v1, int(v2)); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_partner:
			if c = c.partner(sys.bcStack.Pop().ToI(), true); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_enemy:
			if c = c.enemy(sys.bcStack.Pop().ToI()); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_enemynear:
			if c = c.enemyNearTrigger(sys.bcStack.Pop().ToI()); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_playerid:
			if c = sys.playerID(sys.bcStack.Pop().ToI()); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_playerindex:
			if c = sys.playerIndex(sys.bcStack.Pop().ToI()); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_p2:
			if c = c.p2(); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_stateowner:
			if c = sys.chars[c.ss.sb.playerNo][0]; c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_helperindex:
			if c = c.helperIndexTrigger(sys.bcStack.Pop().ToI(), true); c != nil {
				i += 4
				continue
			}
			sys.bcStack.Push(BytecodeSF())
			i += int(*(*int32)(unsafe.Pointer(&be[i]))) + 4
		case OC_rdreset:
			// NOP
		case OC_run:
			l := int(*(*int32)(unsafe.Pointer(&be[i])))
			sys.bcStack.Push(be[i+4 : i+4+l].run(c))
			i += 4 + l
		case OC_nordrun:
			l := int(*(*int32)(unsafe.Pointer(&be[i])))
			sys.bcStack.Push(be[i+4 : i+4+l].run(oc))
			i += 4 + l
			continue
		case OC_int8:
			sys.bcStack.PushI(int32(int8(be[i])))
			i++
		case OC_int:
			sys.bcStack.PushI(*(*int32)(unsafe.Pointer(&be[i])))
			i += 4
		case OC_int64:
			sys.bcStack.PushI64(*(*int64)(unsafe.Pointer(&be[i])))
			i += 8
		case OC_float:
			arr := make([]byte, 4)
			arr[0] = byte(be[i])
			arr[1] = byte(be[i+1])
			arr[2] = byte(be[i+2])
			arr[3] = byte(be[i+3])
			flo := Float32frombytes(arr)
			sys.bcStack.PushF(flo)
			i += 4
		case OC_neg:
			be.neg(sys.bcStack.Top())
		case OC_not:
			be.not(sys.bcStack.Top())
		case OC_blnot:
			be.blnot(sys.bcStack.Top())
		case OC_pow:
			v2 := sys.bcStack.Pop()
			be.pow(sys.bcStack.Top(), v2, sys.workingChar.ss.sb.playerNo)
		case OC_mul:
			v2 := sys.bcStack.Pop()
			be.mul(sys.bcStack.Top(), v2)
		case OC_div:
			v2 := sys.bcStack.Pop()
			be.div(sys.bcStack.Top(), v2)
		case OC_mod:
			v2 := sys.bcStack.Pop()
			be.mod(sys.bcStack.Top(), v2)
		case OC_add:
			v2 := sys.bcStack.Pop()
			be.add(sys.bcStack.Top(), v2)
		case OC_sub:
			v2 := sys.bcStack.Pop()
			be.sub(sys.bcStack.Top(), v2)
		case OC_gt:
			v2 := sys.bcStack.Pop()
			be.gt(sys.bcStack.Top(), v2)
		case OC_ge:
			v2 := sys.bcStack.Pop()
			be.ge(sys.bcStack.Top(), v2)
		case OC_lt:
			v2 := sys.bcStack.Pop()
			be.lt(sys.bcStack.Top(), v2)
		case OC_le:
			v2 := sys.bcStack.Pop()
			be.le(sys.bcStack.Top(), v2)
		case OC_eq:
			v2 := sys.bcStack.Pop()
			be.eq(sys.bcStack.Top(), v2)
		case OC_ne:
			v2 := sys.bcStack.Pop()
			be.ne(sys.bcStack.Top(), v2)
		case OC_and:
			v2 := sys.bcStack.Pop()
			be.and(sys.bcStack.Top(), v2)
		case OC_xor:
			v2 := sys.bcStack.Pop()
			be.xor(sys.bcStack.Top(), v2)
		case OC_or:
			v2 := sys.bcStack.Pop()
			be.or(sys.bcStack.Top(), v2)
		case OC_bland:
			v2 := sys.bcStack.Pop()
			be.bland(sys.bcStack.Top(), v2)
		case OC_blxor:
			v2 := sys.bcStack.Pop()
			be.blxor(sys.bcStack.Top(), v2)
		case OC_blor:
			v2 := sys.bcStack.Pop()
			be.blor(sys.bcStack.Top(), v2)
		case OC_abs:
			be.abs(sys.bcStack.Top())
		case OC_exp:
			be.exp(sys.bcStack.Top())
		case OC_ln:
			be.ln(sys.bcStack.Top())
		case OC_log:
			v2 := sys.bcStack.Pop()
			be.log(sys.bcStack.Top(), v2)
		case OC_cos:
			be.cos(sys.bcStack.Top())
		case OC_sin:
			be.sin(sys.bcStack.Top())
		case OC_tan:
			be.tan(sys.bcStack.Top())
		case OC_acos:
			be.acos(sys.bcStack.Top())
		case OC_asin:
			be.asin(sys.bcStack.Top())
		case OC_atan:
			be.atan(sys.bcStack.Top())
		case OC_floor:
			be.floor(sys.bcStack.Top())
		case OC_ceil:
			be.ceil(sys.bcStack.Top())
		case OC_ifelse:
			v3 := sys.bcStack.Pop()
			v2 := sys.bcStack.Pop()
			if sys.bcStack.Top().ToB() {
				*sys.bcStack.Top() = v2
			} else {
				*sys.bcStack.Top() = v3
			}
		case OC_pop:
			sys.bcStack.Pop()
		case OC_dup:
			sys.bcStack.Dup()
		case OC_swap:
			sys.bcStack.Swap()
		case OC_ailevel:
			if !c.asf(ASF_noailevel) {
				sys.bcStack.PushI(int32(c.getAILevel()))
			} else {
				sys.bcStack.PushI(0)
			}
		case OC_alive:
			sys.bcStack.PushB(c.alive())
		case OC_anim:
			sys.bcStack.PushI(c.animNo)
		case OC_animelemno:
			*sys.bcStack.Top() = c.animElemNo(sys.bcStack.Top().ToI())
		case OC_animelemtime:
			*sys.bcStack.Top() = c.animElemTime(sys.bcStack.Top().ToI())
		case OC_animexist:
			*sys.bcStack.Top() = c.animExist(sys.workingChar, *sys.bcStack.Top())
		case OC_animtime:
			sys.bcStack.PushI(c.animTime())
		case OC_backedge:
			sys.bcStack.PushF(c.backEdge() * (c.localscl / oc.localscl))
		case OC_backedgebodydist:
			sys.bcStack.PushI(int32(c.backEdgeBodyDist() * (c.localscl / oc.localscl)))
		case OC_backedgedist:
			sys.bcStack.PushI(int32(c.backEdgeDist() * (c.localscl / oc.localscl)))
		case OC_bottomedge:
			sys.bcStack.PushF(c.bottomEdge() * (c.localscl / oc.localscl))
		case OC_camerapos_x:
			sys.bcStack.PushF(sys.cam.Pos[0] / oc.localscl)
		case OC_camerapos_y:
			sys.bcStack.PushF((sys.cam.Pos[1] + sys.cam.aspectcorrection + sys.cam.zoomanchorcorrection) / oc.localscl)
		case OC_camerazoom:
			sys.bcStack.PushF(sys.cam.Scale)
		case OC_canrecover:
			sys.bcStack.PushB(c.canRecover())
		case OC_command:
			if c.cmd == nil {
				sys.bcStack.PushB(false)
			} else {
				cmdName := sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[i]))]
				redir := c.playerNo
				pno := c.playerNo
				// For a Mugen character, the command position is checked in the redirecting char
				// Recovery command is an exception in that its position is always checked in the final char
				// Note: In Mugen, a character running a negative state will use its own engine version but the localcoord and commands of the state owner
				// The commands part is not fully recreated at the moment, but no issues have come out of it so far
				if cmdName != "recovery" && oc.stWgi().ikemenver[0] == 0 && oc.stWgi().ikemenver[1] == 0 {
					redir = oc.ss.sb.playerNo
					pno = c.ss.sb.playerNo
				}
				cmdPos, ok := c.cmd[redir].Names[cmdName]
				ok = ok && c.command(pno, cmdPos)
				sys.bcStack.PushB(ok)
			}
			i += 4
		case OC_ctrl:
			sys.bcStack.PushB(c.ctrl())
		case OC_facing:
			sys.bcStack.PushI(int32(c.facing))
		case OC_frontedge:
			sys.bcStack.PushF(c.frontEdge() * (c.localscl / oc.localscl))
		case OC_frontedgebodydist:
			sys.bcStack.PushI(int32(c.frontEdgeBodyDist() * (c.localscl / oc.localscl)))
		case OC_frontedgedist:
			sys.bcStack.PushI(int32(c.frontEdgeDist() * (c.localscl / oc.localscl)))
		case OC_gameheight:
			// Optional exception preventing GameHeight from being affected by stage zoom.
			if c.stWgi().mugenver[0] == 1 && c.stWgi().mugenver[1] == 0 &&
				c.gi().constants["default.legacygamedistancespec"] == 1 {
				sys.bcStack.PushF(c.screenHeight())
			} else {
				sys.bcStack.PushF(c.gameHeight())
			}
		case OC_gametime:
			var pfTime int32
			if sys.netConnection != nil {
				pfTime = sys.netConnection.preFightTime
			} else if sys.replayFile != nil {
				pfTime = sys.replayFile.pfTime
			} else {
				pfTime = sys.preFightTime
			}
			sys.bcStack.PushI(sys.gameTime + pfTime)
		case OC_gamewidth:
			// Optional exception preventing GameWidth from being affected by stage zoom.
			if c.stWgi().mugenver[0] == 1 && c.stWgi().mugenver[1] == 0 &&
				c.gi().constants["default.legacygamedistancespec"] == 1 {
				sys.bcStack.PushF(c.screenWidth())
			} else {
				sys.bcStack.PushF(c.gameWidth())
			}
		case OC_hitcount:
			sys.bcStack.PushI(c.hitCount)
		case OC_hitdefattr:
			sys.bcStack.PushB(c.hitDefAttr(*(*int32)(unsafe.Pointer(&be[i]))))
			i += 4
		case OC_hitfall:
			sys.bcStack.PushB(c.ghv.fallflag)
		case OC_hitover:
			sys.bcStack.PushB(c.hitOver())
		case OC_hitpausetime:
			sys.bcStack.PushI(c.hitPauseTime)
		case OC_hitshakeover:
			sys.bcStack.PushB(c.hitShakeOver())
		case OC_hitvel_x:
			// This trigger is bugged in Mugen 1.1, with its output being affected by game resolution
			sys.bcStack.PushF(c.ghv.xvel * c.facing * (c.localscl / oc.localscl))
		case OC_hitvel_y:
			sys.bcStack.PushF(c.ghv.yvel * (c.localscl / oc.localscl))
		case OC_hitvel_z:
			sys.bcStack.PushF(c.ghv.zvel * (c.localscl / oc.localscl))
		case OC_id:
			sys.bcStack.PushI(c.id)
		case OC_inguarddist:
			sys.bcStack.PushB(c.inguarddist)
		case OC_ishelper:
			index := sys.bcStack.Pop().ToI()
			id := sys.bcStack.Pop().ToI()
			sys.bcStack.PushB(c.isHelper(id, int(index)))
		case OC_leftedge:
			sys.bcStack.PushF(c.leftEdge() * (c.localscl / oc.localscl))
		case OC_life:
			sys.bcStack.PushI(c.life)
		case OC_lifemax:
			sys.bcStack.PushI(c.lifeMax)
		case OC_movecontact:
			sys.bcStack.PushI(c.moveContact())
		case OC_moveguarded:
			sys.bcStack.PushI(c.moveGuarded())
		case OC_movehit:
			sys.bcStack.PushI(c.moveHit())
		case OC_movereversed:
			sys.bcStack.PushI(c.moveReversed())
		case OC_movetype:
			sys.bcStack.PushB(c.ss.moveType == MoveType(be[i])<<15)
			i++
		case OC_numenemy:
			sys.bcStack.PushI(c.numEnemy())
		case OC_numexplod:
			*sys.bcStack.Top() = c.numExplod(*sys.bcStack.Top())
		case OC_numhelper:
			*sys.bcStack.Top() = c.numHelper(*sys.bcStack.Top())
		case OC_numpartner:
			sys.bcStack.PushI(c.numPartner())
		case OC_numproj:
			sys.bcStack.PushI(c.numProj())
		case OC_numprojid:
			*sys.bcStack.Top() = c.numProjID(*sys.bcStack.Top())
		case OC_numtarget:
			*sys.bcStack.Top() = c.numTarget(*sys.bcStack.Top())
		case OC_numtext:
			*sys.bcStack.Top() = c.numText(*sys.bcStack.Top())
		case OC_palno:
			sys.bcStack.PushI(c.gi().palno)
			// In Winmugen a helper's PalNo is always 1
			// That behavior has no apparent benefits and even Mugen 1.0 compatibility mode does not keep it
		case OC_pos_x:
			sys.bcStack.PushF((c.pos[0]*(c.localscl/oc.localscl) - sys.cam.Pos[0]/oc.localscl))
		case OC_pos_y:
			sys.bcStack.PushF((c.pos[1] - c.groundLevel - c.platformPosY) * (c.localscl / oc.localscl))
		case OC_power:
			sys.bcStack.PushI(c.getPower())
		case OC_powermax:
			sys.bcStack.PushI(c.powerMax)
		case OC_playeridexist:
			*sys.bcStack.Top() = sys.playerIDExist(*sys.bcStack.Top())
		case OC_prevstateno:
			sys.bcStack.PushI(c.ss.prevno)
		case OC_projcanceltime:
			*sys.bcStack.Top() = c.projCancelTime(*sys.bcStack.Top())
		case OC_projcontacttime:
			*sys.bcStack.Top() = c.projContactTime(*sys.bcStack.Top())
		case OC_projguardedtime:
			*sys.bcStack.Top() = c.projGuardedTime(*sys.bcStack.Top())
		case OC_projhittime:
			*sys.bcStack.Top() = c.projHitTime(*sys.bcStack.Top())
		case OC_random:
			sys.bcStack.PushI(Rand(0, 999))
		case OC_rightedge:
			sys.bcStack.PushF(c.rightEdge() * (c.localscl / oc.localscl))
		case OC_roundstate:
			sys.bcStack.PushI(sys.roundState())
		case OC_roundswon:
			sys.bcStack.PushI(c.roundsWon())
		case OC_screenheight:
			sys.bcStack.PushF(c.screenHeight())
		case OC_screenpos_x:
			sys.bcStack.PushF(c.screenPosX() / oc.localscl)
		case OC_screenpos_y:
			sys.bcStack.PushF(c.screenPosY() / oc.localscl)
		case OC_screenwidth:
			sys.bcStack.PushF(c.screenWidth())
		case OC_selfanimexist:
			*sys.bcStack.Top() = c.selfAnimExist(*sys.bcStack.Top())
		case OC_stateno:
			sys.bcStack.PushI(c.ss.no)
		case OC_statetype:
			sys.bcStack.PushB(c.ss.stateType == StateType(be[i]))
			i++
		case OC_teammode:
			if c.teamside == -1 {
				sys.bcStack.PushB(TM_Single == TeamMode(be[i]))
			} else {
				sys.bcStack.PushB(sys.tmode[c.playerNo&1] == TeamMode(be[i]))
			}
			i++
		case OC_teamside:
			sys.bcStack.PushI(int32(c.teamside) + 1)
		case OC_time:
			sys.bcStack.PushI(c.time())
		case OC_topedge:
			sys.bcStack.PushF(c.topEdge() * (c.localscl / oc.localscl))
		case OC_uniqhitcount:
			sys.bcStack.PushI(c.uniqHitCount)
		case OC_vel_x:
			sys.bcStack.PushF(c.vel[0] * (c.localscl / oc.localscl))
		case OC_vel_y:
			sys.bcStack.PushF(c.vel[1] * (c.localscl / oc.localscl))
		case OC_vel_z:
			sys.bcStack.PushF(c.vel[2] * (c.localscl / oc.localscl))
		case OC_st_:
			be.run_st(c, &i)
		case OC_const_:
			be.run_const(c, &i, oc)
		case OC_ex_:
			be.run_ex(c, &i, oc)
		case OC_ex2_:
			be.run_ex2(c, &i, oc)
		case OC_var:
			*sys.bcStack.Top() = c.varGet(sys.bcStack.Top().ToI())
		case OC_sysvar:
			*sys.bcStack.Top() = c.sysVarGet(sys.bcStack.Top().ToI())
		case OC_fvar:
			*sys.bcStack.Top() = c.fvarGet(sys.bcStack.Top().ToI())
		case OC_sysfvar:
			*sys.bcStack.Top() = c.sysFvarGet(sys.bcStack.Top().ToI())
		case OC_localvar:
			sys.bcStack.Push(sys.bcVar[uint8(be[i])])
			i++
		}
		c = oc
	}
	return sys.bcStack.Pop()
}

func (be BytecodeExp) run_st(c *Char, i *int) {
	(*i)++
	opc := be[*i-1]
	switch opc {
	case OC_st_var:
		v := sys.bcStack.Pop().ToI()
		*sys.bcStack.Top() = c.varSet(sys.bcStack.Top().ToI(), v)
	case OC_st_sysvar:
		v := sys.bcStack.Pop().ToI()
		*sys.bcStack.Top() = c.sysVarSet(sys.bcStack.Top().ToI(), v)
	case OC_st_fvar:
		v := sys.bcStack.Pop().ToF()
		*sys.bcStack.Top() = c.fvarSet(sys.bcStack.Top().ToI(), v)
	case OC_st_sysfvar:
		v := sys.bcStack.Pop().ToF()
		*sys.bcStack.Top() = c.sysFvarSet(sys.bcStack.Top().ToI(), v)
	case OC_st_varadd:
		v := sys.bcStack.Pop().ToI()
		*sys.bcStack.Top() = c.varAdd(sys.bcStack.Top().ToI(), v)
	case OC_st_sysvaradd:
		v := sys.bcStack.Pop().ToI()
		*sys.bcStack.Top() = c.sysVarAdd(sys.bcStack.Top().ToI(), v)
	case OC_st_fvaradd:
		v := sys.bcStack.Pop().ToF()
		*sys.bcStack.Top() = c.fvarAdd(sys.bcStack.Top().ToI(), v)
	case OC_st_sysfvaradd:
		v := sys.bcStack.Pop().ToF()
		*sys.bcStack.Top() = c.sysFvarAdd(sys.bcStack.Top().ToI(), v)
	case OC_st_map:
		v := sys.bcStack.Pop().ToF()
		sys.bcStack.Push(c.mapSet(sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))], v, 0))
		*i += 4
	}
}

func (be BytecodeExp) run_const(c *Char, i *int, oc *Char) {
	(*i)++
	opc := be[*i-1]
	switch opc {
	case OC_const_data_life:
		sys.bcStack.PushI(c.gi().data.life)
	case OC_const_data_power:
		sys.bcStack.PushI(c.gi().data.power)
	case OC_const_data_dizzypoints:
		sys.bcStack.PushI(c.gi().data.dizzypoints)
	case OC_const_data_guardpoints:
		sys.bcStack.PushI(c.gi().data.guardpoints)
	case OC_const_data_attack:
		sys.bcStack.PushI(c.gi().data.attack)
	case OC_const_data_defence:
		sys.bcStack.PushI(c.gi().data.defence)
	case OC_const_data_fall_defence_up:
		sys.bcStack.PushI(c.gi().data.fall.defence_up)
	case OC_const_data_fall_defence_mul:
		sys.bcStack.PushF(1.0 / c.gi().data.fall.defence_mul)
	case OC_const_data_liedown_time:
		sys.bcStack.PushI(c.gi().data.liedown.time)
	case OC_const_data_airjuggle:
		sys.bcStack.PushI(c.gi().data.airjuggle)
	case OC_const_data_sparkno:
		sys.bcStack.PushI(c.gi().data.sparkno)
	case OC_const_data_guard_sparkno:
		sys.bcStack.PushI(c.gi().data.guard.sparkno)
	case OC_const_data_hitsound_channel:
		sys.bcStack.PushI(c.gi().data.hitsound_channel)
	case OC_const_data_guardsound_channel:
		sys.bcStack.PushI(c.gi().data.guardsound_channel)
	case OC_const_data_ko_echo:
		sys.bcStack.PushI(c.gi().data.ko.echo)
	case OC_const_data_volume:
		sys.bcStack.PushI(c.gi().data.volume)
	case OC_const_data_intpersistindex:
		sys.bcStack.PushI(c.gi().data.intpersistindex)
	case OC_const_data_floatpersistindex:
		sys.bcStack.PushI(c.gi().data.floatpersistindex)
	case OC_const_size_xscale:
		sys.bcStack.PushF(c.size.xscale)
	case OC_const_size_yscale:
		sys.bcStack.PushF(c.size.yscale)
	case OC_const_size_ground_back:
		sys.bcStack.PushF(c.size.ground.back * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_ground_front:
		sys.bcStack.PushF(c.size.ground.front * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_air_back:
		sys.bcStack.PushF(c.size.air.back * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_air_front:
		sys.bcStack.PushF(c.size.air.front * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_height_stand:
		sys.bcStack.PushF(c.size.height.stand * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_height_crouch:
		sys.bcStack.PushF(c.size.height.crouch * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_height_air_top:
		sys.bcStack.PushF(c.size.height.air[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_height_air_bottom:
		sys.bcStack.PushF(c.size.height.air[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_height_down:
		sys.bcStack.PushF(c.size.height.down * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_dist_width_front:
		sys.bcStack.PushF(c.size.attack.dist.width[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_dist_width_back:
		sys.bcStack.PushF(c.size.attack.dist.width[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_dist_height_top:
		sys.bcStack.PushF(c.size.attack.dist.height[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_dist_height_bottom:
		sys.bcStack.PushF(c.size.attack.dist.height[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_dist_depth_top:
		sys.bcStack.PushF(c.size.attack.dist.depth[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_dist_depth_bottom:
		sys.bcStack.PushF(c.size.attack.dist.depth[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_depth_top:
		sys.bcStack.PushF(c.size.attack.depth[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_attack_depth_bottom:
		sys.bcStack.PushF(c.size.attack.depth[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_proj_attack_dist_width_front:
		sys.bcStack.PushF(c.size.proj.attack.dist.width[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_proj_attack_dist_width_back:
		sys.bcStack.PushF(c.size.proj.attack.dist.width[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_proj_attack_dist_height_top:
		sys.bcStack.PushF(c.size.proj.attack.dist.height[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_proj_attack_dist_height_bottom:
		sys.bcStack.PushF(c.size.proj.attack.dist.height[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_proj_attack_dist_depth_top:
		sys.bcStack.PushF(c.size.proj.attack.dist.depth[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_proj_attack_dist_depth_bottom:
		sys.bcStack.PushF(c.size.proj.attack.dist.depth[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_proj_doscale:
		sys.bcStack.PushI(c.size.proj.doscale)
	case OC_const_size_head_pos_x:
		sys.bcStack.PushF(c.size.head.pos[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_head_pos_y:
		sys.bcStack.PushF(c.size.head.pos[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_mid_pos_x:
		sys.bcStack.PushF(c.size.mid.pos[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_mid_pos_y:
		sys.bcStack.PushF(c.size.mid.pos[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_shadowoffset:
		sys.bcStack.PushF(c.size.shadowoffset * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_draw_offset_x:
		sys.bcStack.PushF(c.size.draw.offset[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_draw_offset_y:
		sys.bcStack.PushF(c.size.draw.offset[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_depth_top:
		sys.bcStack.PushF(c.size.depth[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_depth_bottom:
		sys.bcStack.PushF(c.size.depth[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_size_weight:
		sys.bcStack.PushI(c.size.weight)
	case OC_const_size_pushfactor:
		sys.bcStack.PushF(c.size.pushfactor)
	case OC_const_velocity_air_gethit_airrecover_add_x:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.add[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_airrecover_add_y:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.add[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_airrecover_back:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.back * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_airrecover_down:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.down * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_airrecover_fwd:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.fwd * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_airrecover_mul_x:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.mul[0])
	case OC_const_velocity_air_gethit_airrecover_mul_y:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.mul[1])
	case OC_const_velocity_air_gethit_airrecover_up:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.airrecover.up * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_groundrecover_x:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.groundrecover[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_groundrecover_y:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.groundrecover[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_ko_add_x:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.ko.add[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_ko_add_y:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.ko.add[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_air_gethit_ko_ymin:
		sys.bcStack.PushF(c.gi().velocity.air.gethit.ko.ymin * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_back_x:
		sys.bcStack.PushF(c.gi().velocity.airjump.back * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_down_x:
		sys.bcStack.PushF(c.gi().velocity.airjump.down[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_down_y:
		sys.bcStack.PushF(c.gi().velocity.airjump.down[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_down_z:
		sys.bcStack.PushF(c.gi().velocity.airjump.down[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_fwd_x:
		sys.bcStack.PushF(c.gi().velocity.airjump.fwd * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_neu_x:
		sys.bcStack.PushF(c.gi().velocity.airjump.neu[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_up_x:
		sys.bcStack.PushF(c.gi().velocity.airjump.up[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_up_y:
		sys.bcStack.PushF(c.gi().velocity.airjump.up[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_up_z:
		sys.bcStack.PushF(c.gi().velocity.airjump.up[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_airjump_y:
		sys.bcStack.PushF(c.gi().velocity.airjump.neu[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_ground_gethit_ko_add_x:
		sys.bcStack.PushF(c.gi().velocity.ground.gethit.ko.add[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_ground_gethit_ko_add_y:
		sys.bcStack.PushF(c.gi().velocity.ground.gethit.ko.add[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_ground_gethit_ko_xmul:
		sys.bcStack.PushF(c.gi().velocity.ground.gethit.ko.xmul)
	case OC_const_velocity_ground_gethit_ko_ymin:
		sys.bcStack.PushF(c.gi().velocity.ground.gethit.ko.ymin * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_back_x:
		sys.bcStack.PushF(c.gi().velocity.jump.back * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_down_x:
		sys.bcStack.PushF(c.gi().velocity.jump.down[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_down_y:
		sys.bcStack.PushF(c.gi().velocity.jump.down[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_down_z:
		sys.bcStack.PushF(c.gi().velocity.jump.down[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_fwd_x:
		sys.bcStack.PushF(c.gi().velocity.jump.fwd * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_neu_x:
		sys.bcStack.PushF(c.gi().velocity.jump.neu[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_up_x:
		sys.bcStack.PushF(c.gi().velocity.jump.up[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_up_y:
		sys.bcStack.PushF(c.gi().velocity.jump.up[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_up_z:
		sys.bcStack.PushF(c.gi().velocity.jump.up[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_jump_y:
		sys.bcStack.PushF(c.gi().velocity.jump.neu[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_back_x:
		sys.bcStack.PushF(c.gi().velocity.run.back[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_back_y:
		sys.bcStack.PushF(c.gi().velocity.run.back[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_down_x:
		sys.bcStack.PushF(c.gi().velocity.run.down[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_down_y:
		sys.bcStack.PushF(c.gi().velocity.run.down[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_down_z:
		sys.bcStack.PushF(c.gi().velocity.run.down[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_fwd_x:
		sys.bcStack.PushF(c.gi().velocity.run.fwd[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_fwd_y:
		sys.bcStack.PushF(c.gi().velocity.run.fwd[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_up_x:
		sys.bcStack.PushF(c.gi().velocity.run.up[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_up_y:
		sys.bcStack.PushF(c.gi().velocity.run.up[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_run_up_z:
		sys.bcStack.PushF(c.gi().velocity.run.up[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_back_x:
		sys.bcStack.PushF(c.gi().velocity.runjump.back[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_back_y:
		sys.bcStack.PushF(c.gi().velocity.runjump.back[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_down_x:
		sys.bcStack.PushF(c.gi().velocity.runjump.down[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_down_y:
		sys.bcStack.PushF(c.gi().velocity.runjump.down[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_down_z:
		sys.bcStack.PushF(c.gi().velocity.runjump.down[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_fwd_x:
		sys.bcStack.PushF(c.gi().velocity.runjump.fwd[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_up_x:
		sys.bcStack.PushF(c.gi().velocity.runjump.up[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_up_y:
		sys.bcStack.PushF(c.gi().velocity.runjump.up[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_up_z:
		sys.bcStack.PushF(c.gi().velocity.runjump.up[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_runjump_y:
		sys.bcStack.PushF(c.gi().velocity.runjump.fwd[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_back_x:
		sys.bcStack.PushF(c.gi().velocity.walk.back * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_down_x:
		sys.bcStack.PushF(c.gi().velocity.walk.down[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_down_y:
		sys.bcStack.PushF(c.gi().velocity.walk.down[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_down_z:
		sys.bcStack.PushF(c.gi().velocity.walk.down[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_fwd_x:
		sys.bcStack.PushF(c.gi().velocity.walk.fwd * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_up_x:
		sys.bcStack.PushF(c.gi().velocity.walk.up[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_up_y:
		sys.bcStack.PushF(c.gi().velocity.walk.up[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_velocity_walk_up_z:
		sys.bcStack.PushF(c.gi().velocity.walk.up[2] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_airjump_num:
		sys.bcStack.PushI(c.gi().movement.airjump.num)
	case OC_const_movement_airjump_height:
		sys.bcStack.PushF(c.gi().movement.airjump.height * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_yaccel:
		sys.bcStack.PushF(c.gi().movement.yaccel * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_stand_friction:
		sys.bcStack.PushF(c.gi().movement.stand.friction)
	case OC_const_movement_crouch_friction:
		sys.bcStack.PushF(c.gi().movement.crouch.friction)
	case OC_const_movement_stand_friction_threshold:
		sys.bcStack.PushF(c.gi().movement.stand.friction_threshold * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_crouch_friction_threshold:
		sys.bcStack.PushF(c.gi().movement.crouch.friction_threshold * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_air_gethit_groundlevel:
		sys.bcStack.PushF(c.gi().movement.air.gethit.groundlevel * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_air_gethit_groundrecover_ground_threshold:
		sys.bcStack.PushF(
			c.gi().movement.air.gethit.groundrecover.ground.threshold * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_air_gethit_groundrecover_groundlevel:
		sys.bcStack.PushF(c.gi().movement.air.gethit.groundrecover.groundlevel * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_air_gethit_airrecover_threshold:
		sys.bcStack.PushF(c.gi().movement.air.gethit.airrecover.threshold * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_air_gethit_airrecover_yaccel:
		sys.bcStack.PushF(c.gi().movement.air.gethit.airrecover.yaccel * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_air_gethit_trip_groundlevel:
		sys.bcStack.PushF(c.gi().movement.air.gethit.trip.groundlevel * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_down_bounce_offset_x:
		sys.bcStack.PushF(c.gi().movement.down.bounce.offset[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_down_bounce_offset_y:
		sys.bcStack.PushF(c.gi().movement.down.bounce.offset[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_down_bounce_yaccel:
		sys.bcStack.PushF(c.gi().movement.down.bounce.yaccel * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_down_bounce_groundlevel:
		sys.bcStack.PushF(c.gi().movement.down.bounce.groundlevel * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_down_gethit_offset_x:
		sys.bcStack.PushF(c.gi().movement.down.gethit.offset[0] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_down_gethit_offset_y:
		sys.bcStack.PushF(c.gi().movement.down.gethit.offset[1] * ((320 / c.localcoord) / oc.localscl))
	case OC_const_movement_down_friction_threshold:
		sys.bcStack.PushF(c.gi().movement.down.friction_threshold * ((320 / c.localcoord) / oc.localscl))
	case OC_const_authorname:
		sys.bcStack.PushB(c.gi().authorLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_displayname:
		sys.bcStack.PushB(c.gi().displaynameLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_name:
		sys.bcStack.PushB(c.gi().nameLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_p2name:
		p2 := c.p2()
		sys.bcStack.PushB(p2 != nil &&
			p2.gi().nameLow == sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_p3name:
		p3 := c.partner(0, false)
		sys.bcStack.PushB(p3 != nil &&
			p3.gi().nameLow == sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_p4name:
		p4 := sys.charList.enemyNear(c, 1, true, false)
		sys.bcStack.PushB(p4 != nil &&
			p4.gi().nameLow == sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_p5name:
		p5 := c.partner(1, false)
		sys.bcStack.PushB(p5 != nil &&
			p5.gi().nameLow == sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_p6name:
		p6 := sys.charList.enemyNear(c, 2, true, false)
		sys.bcStack.PushB(p6 != nil &&
			p6.gi().nameLow == sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_p7name:
		p7 := c.partner(2, false)
		sys.bcStack.PushB(p7 != nil &&
			p7.gi().nameLow == sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_p8name:
		p8 := sys.charList.enemyNear(c, 3, true, false)
		sys.bcStack.PushB(p8 != nil &&
			p8.gi().nameLow == sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	// StageVar
	case OC_const_stagevar_info_name:
		sys.bcStack.PushB(sys.stage.nameLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_stagevar_info_displayname:
		sys.bcStack.PushB(sys.stage.displaynameLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_stagevar_info_author:
		sys.bcStack.PushB(sys.stage.authorLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_const_stagevar_camera_autocenter:
		sys.bcStack.PushB(sys.stage.stageCamera.autocenter)
	case OC_const_stagevar_camera_boundleft:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.boundleft) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_boundright:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.boundright) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_boundhigh:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.boundhigh) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_boundlow:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.boundlow) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_floortension:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.floortension) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_lowestcap:
		sys.bcStack.PushB(sys.stage.stageCamera.lowestcap)
	case OC_const_stagevar_camera_tension:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.tension) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_tensionvel:
		sys.bcStack.PushF(sys.stage.stageCamera.tensionvel)
	case OC_const_stagevar_camera_cuthigh:
		sys.bcStack.PushI(sys.stage.stageCamera.cuthigh)
	case OC_const_stagevar_camera_cutlow:
		sys.bcStack.PushI(sys.stage.stageCamera.cutlow)
	case OC_const_stagevar_camera_tensionhigh:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.tensionhigh) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_tensionlow:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.tensionlow) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_camera_startzoom:
		sys.bcStack.PushF(sys.stage.stageCamera.startzoom)
	case OC_const_stagevar_camera_verticalfollow:
		sys.bcStack.PushF(sys.stage.stageCamera.verticalfollow)
	case OC_const_stagevar_camera_zoomout:
		sys.bcStack.PushF(sys.stage.stageCamera.zoomout)
	case OC_const_stagevar_camera_zoomin:
		sys.bcStack.PushF(sys.stage.stageCamera.zoomin)
	case OC_const_stagevar_camera_zoomindelay:
		sys.bcStack.PushF(sys.stage.stageCamera.zoomindelay)
	case OC_const_stagevar_camera_zoominspeed:
		sys.bcStack.PushF(sys.stage.stageCamera.zoominspeed)
	case OC_const_stagevar_camera_zoomoutspeed:
		sys.bcStack.PushF(sys.stage.stageCamera.zoomoutspeed)
	case OC_const_stagevar_camera_yscrollspeed:
		sys.bcStack.PushF(sys.stage.stageCamera.yscrollspeed)
	case OC_const_stagevar_camera_ytension_enable:
		sys.bcStack.PushB(sys.stage.stageCamera.ytensionenable)
	case OC_const_stagevar_playerinfo_leftbound:
		sys.bcStack.PushF(sys.stage.leftbound * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_playerinfo_rightbound:
		sys.bcStack.PushF(sys.stage.rightbound * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_playerinfo_topbound:
		sys.bcStack.PushF(sys.stage.topbound * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_playerinfo_botbound:
		sys.bcStack.PushF(sys.stage.botbound * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_scaling_topz:
		sys.bcStack.PushF(sys.stage.stageCamera.topz)
	case OC_const_stagevar_scaling_botz:
		sys.bcStack.PushF(sys.stage.stageCamera.botz)
	case OC_const_stagevar_scaling_topscale:
		sys.bcStack.PushF(sys.stage.stageCamera.ztopscale)
	case OC_const_stagevar_scaling_botscale:
		sys.bcStack.PushF(sys.stage.stageCamera.zbotscale)
	case OC_const_stagevar_bound_screenleft:
		sys.bcStack.PushI(int32(float32(sys.stage.screenleft) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_bound_screenright:
		sys.bcStack.PushI(int32(float32(sys.stage.screenright) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_stageinfo_localcoord_x:
		sys.bcStack.PushI(sys.stage.stageCamera.localcoord[0])
	case OC_const_stagevar_stageinfo_localcoord_y:
		sys.bcStack.PushI(sys.stage.stageCamera.localcoord[1])
	case OC_const_stagevar_stageinfo_xscale:
		sys.bcStack.PushF(sys.stage.scale[0])
	case OC_const_stagevar_stageinfo_yscale:
		sys.bcStack.PushF(sys.stage.scale[1])
	case OC_const_stagevar_stageinfo_zoffset:
		sys.bcStack.PushI(int32(float32(sys.stage.stageCamera.zoffset) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_stageinfo_zoffsetlink:
		sys.bcStack.PushI(sys.stage.zoffsetlink)
	case OC_const_stagevar_shadow_intensity:
		sys.bcStack.PushI(sys.stage.sdw.intensity)
	case OC_const_stagevar_shadow_color_r:
		sys.bcStack.PushI(int32((sys.stage.sdw.color & 0xFF0000) >> 16))
	case OC_const_stagevar_shadow_color_g:
		sys.bcStack.PushI(int32((sys.stage.sdw.color & 0xFF00) >> 8))
	case OC_const_stagevar_shadow_color_b:
		sys.bcStack.PushI(int32(sys.stage.sdw.color & 0xFF))
	case OC_const_stagevar_shadow_yscale:
		sys.bcStack.PushF(sys.stage.sdw.yscale)
	case OC_const_stagevar_shadow_fade_range_begin:
		sys.bcStack.PushI(int32(float32(sys.stage.sdw.fadebgn) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_shadow_fade_range_end:
		sys.bcStack.PushI(int32(float32(sys.stage.sdw.fadeend) * sys.stage.localscl / oc.localscl))
	case OC_const_stagevar_shadow_xshear:
		sys.bcStack.PushF(sys.stage.sdw.xshear)
	case OC_const_stagevar_shadow_offset_x:
		sys.bcStack.PushF(sys.stage.sdw.offset[0] * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_shadow_offset_y:
		sys.bcStack.PushF(sys.stage.sdw.offset[1] * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_reflection_intensity:
		sys.bcStack.PushI(sys.stage.reflection.intensity)
	case OC_const_stagevar_reflection_yscale:
		sys.bcStack.PushF(sys.stage.reflection.yscale)
	case OC_const_stagevar_reflection_offset_x:
		sys.bcStack.PushF(sys.stage.reflection.offset[0] * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_reflection_offset_y:
		sys.bcStack.PushF(sys.stage.reflection.offset[1] * sys.stage.localscl / oc.localscl)
	case OC_const_stagevar_reflection_xshear:
		sys.bcStack.PushF(sys.stage.reflection.xshear)
	case OC_const_stagevar_reflection_color_r:
		sys.bcStack.PushI(int32((sys.stage.reflection.color & 0xFF0000) >> 16))
	case OC_const_stagevar_reflection_color_g:
		sys.bcStack.PushI(int32((sys.stage.reflection.color & 0xFF00) >> 8))
	case OC_const_stagevar_reflection_color_b:
		sys.bcStack.PushI(int32(sys.stage.reflection.color & 0xFF))
	case OC_const_gameoption:
		value, err := sys.cfg.GetValue(sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
			unsafe.Pointer(&be[*i]))])
		if err == nil {
			switch v := value.(type) {
			case bool:
				sys.bcStack.PushB(v)
			case float32:
				sys.bcStack.PushF(v)
			case float64:
				sys.bcStack.PushF(float32(v))
			case int:
				sys.bcStack.PushI(int32(v))
			case int64:
				sys.bcStack.PushI(int32(v))
			case int32:
				sys.bcStack.PushI(v)
			default:
				sys.bcStack.PushB(false)
			}
		} else {
			sys.bcStack.PushB(false)
		}
		*i += 4
	case OC_const_constants:
		sys.bcStack.PushF(c.gi().constants[sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
			unsafe.Pointer(&be[*i]))]])
		*i += 4
	case OC_const_stage_constants:
		sys.bcStack.PushF(sys.stage.constants[sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
			unsafe.Pointer(&be[*i]))]])
		*i += 4
	default:
		sys.errLog.Printf("%v\n", be[*i-1])
		c.panic()
	}
}

func (be BytecodeExp) run_ex(c *Char, i *int, oc *Char) {
	(*i)++
	opc := be[*i-1]
	switch opc {
	case OC_ex_p2dist_x:
		sys.bcStack.Push(c.rdDistX(c.p2(), oc))
	case OC_ex_p2dist_y:
		sys.bcStack.Push(c.rdDistY(c.p2(), oc))
	case OC_ex_p2dist_z:
		sys.bcStack.Push(c.rdDistZ(c.p2(), oc))
	case OC_ex_p2bodydist_x:
		sys.bcStack.Push(c.p2BodyDistX(oc))
	case OC_ex_p2bodydist_y:
		sys.bcStack.Push(c.p2BodyDistY(oc))
	case OC_ex_p2bodydist_z:
		sys.bcStack.Push(c.p2BodyDistZ(oc))
	case OC_ex_parentdist_x:
		sys.bcStack.Push(c.rdDistX(c.parent(true), oc))
	case OC_ex_parentdist_y:
		sys.bcStack.Push(c.rdDistY(c.parent(true), oc))
	case OC_ex_parentdist_z:
		sys.bcStack.Push(c.rdDistZ(c.parent(true), oc))
	case OC_ex_rootdist_x:
		sys.bcStack.Push(c.rdDistX(c.root(true), oc))
	case OC_ex_rootdist_y:
		sys.bcStack.Push(c.rdDistY(c.root(true), oc))
	case OC_ex_rootdist_z:
		sys.bcStack.Push(c.rdDistZ(c.root(true), oc))
	case OC_ex_win:
		sys.bcStack.PushB(c.win())
	case OC_ex_winko:
		sys.bcStack.PushB(c.winKO())
	case OC_ex_wintime:
		sys.bcStack.PushB(c.winTime())
	case OC_ex_winperfect:
		sys.bcStack.PushB(c.winPerfect())
	case OC_ex_winspecial:
		sys.bcStack.PushB(c.winType(WT_Special))
	case OC_ex_winhyper:
		sys.bcStack.PushB(c.winType(WT_Hyper))
	case OC_ex_lose:
		sys.bcStack.PushB(c.lose())
	case OC_ex_loseko:
		sys.bcStack.PushB(c.loseKO())
	case OC_ex_losetime:
		sys.bcStack.PushB(c.loseTime())
	case OC_ex_drawgame:
		sys.bcStack.PushB(c.drawgame())
	case OC_ex_matchover:
		sys.bcStack.PushB(sys.matchOver())
	case OC_ex_matchno:
		sys.bcStack.PushI(sys.match)
	case OC_ex_roundno:
		sys.bcStack.PushI(sys.round)
	case OC_ex_roundsexisted:
		sys.bcStack.PushI(c.roundsExisted())
	case OC_ex_ishometeam:
		sys.bcStack.PushB(c.teamside == sys.home)
	case OC_ex_tickspersecond:
		sys.bcStack.PushI(int32((60 + sys.cfg.Options.GameSpeed*5) * sys.accel))
	case OC_ex_const240p:
		*sys.bcStack.Top() = c.constp(320, sys.bcStack.Top().ToF())
	case OC_ex_const480p:
		*sys.bcStack.Top() = c.constp(640, sys.bcStack.Top().ToF())
	case OC_ex_const720p:
		*sys.bcStack.Top() = c.constp(1280, sys.bcStack.Top().ToF())
	case OC_ex_const1080p:
		*sys.bcStack.Top() = c.constp(1920, sys.bcStack.Top().ToF())
	case OC_ex_gethitvar_animtype:
		sys.bcStack.PushI(int32(c.ghv.animtype))
	case OC_ex_gethitvar_air_animtype:
		sys.bcStack.PushI(int32(c.ghv.airanimtype))
	case OC_ex_gethitvar_ground_animtype:
		sys.bcStack.PushI(int32(c.ghv.groundanimtype))
	case OC_ex_gethitvar_fall_animtype:
		sys.bcStack.PushI(int32(c.ghv.fall_animtype))
	case OC_ex_gethitvar_type:
		sys.bcStack.PushI(int32(c.ghv._type))
	case OC_ex_gethitvar_airtype:
		sys.bcStack.PushI(int32(c.ghv.airtype))
	case OC_ex_gethitvar_groundtype:
		sys.bcStack.PushI(int32(c.ghv.groundtype))
	case OC_ex_gethitvar_damage:
		sys.bcStack.PushI(c.ghv.damage)
	case OC_ex_gethitvar_guardcount:
		sys.bcStack.PushI(c.ghv.guardcount)
	case OC_ex_gethitvar_hitcount:
		sys.bcStack.PushI(c.ghv.hitcount)
	case OC_ex_gethitvar_fallcount:
		sys.bcStack.PushI(c.ghv.fallcount)
	case OC_ex_gethitvar_hitshaketime:
		sys.bcStack.PushI(c.ghv.hitshaketime)
	case OC_ex_gethitvar_hittime:
		sys.bcStack.PushI(c.ghv.hittime)
	case OC_ex_gethitvar_slidetime:
		sys.bcStack.PushI(c.ghv.slidetime)
	case OC_ex_gethitvar_ctrltime:
		sys.bcStack.PushI(c.ghv.ctrltime)
	case OC_ex_gethitvar_down_recovertime:
		sys.bcStack.PushI(c.ghv.down_recovertime)
	case OC_ex_gethitvar_xoff:
		sys.bcStack.PushF(c.ghv.xoff * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_yoff:
		sys.bcStack.PushF(c.ghv.yoff * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_zoff:
		sys.bcStack.PushF(c.ghv.zoff * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_xvel:
		sys.bcStack.PushF(c.ghv.xvel * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_yvel:
		sys.bcStack.PushF(c.ghv.yvel * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_zvel:
		sys.bcStack.PushF(c.ghv.zvel * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_xaccel:
		sys.bcStack.PushF(c.ghv.xaccel * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_yaccel:
		sys.bcStack.PushF(c.ghv.yaccel * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_zaccel:
		sys.bcStack.PushF(c.ghv.zaccel * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_xveladd:
		sys.bcStack.PushF(c.ghv.xveladd * (c.localscl / oc.localscl)) // Mugen has these two apparently dummied out
	case OC_ex_gethitvar_yveladd:
		sys.bcStack.PushF(c.ghv.yveladd * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_chainid:
		sys.bcStack.PushI(c.ghv.chainId())
	case OC_ex_gethitvar_guarded:
		sys.bcStack.PushB(c.ghv.guarded)
	case OC_ex_gethitvar_isbound:
		sys.bcStack.PushB(c.isTargetBound())
	case OC_ex_gethitvar_fall:
		sys.bcStack.PushB(c.ghv.fallflag)
	case OC_ex_gethitvar_fall_damage:
		sys.bcStack.PushI(c.ghv.fall_damage)
	case OC_ex_gethitvar_fall_xvel:
		if math.IsNaN(float64(c.ghv.fall_xvelocity)) {
			sys.bcStack.PushF(-32760) // Winmugen behavior
		} else {
			sys.bcStack.PushF(c.ghv.fall_xvelocity * (c.localscl / oc.localscl))
		}
	case OC_ex_gethitvar_fall_yvel:
		sys.bcStack.PushF(c.ghv.fall_yvelocity * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_fall_zvel:
		if math.IsNaN(float64(c.ghv.fall_zvelocity)) {
			sys.bcStack.PushF(-32760) // Winmugen behavior
		} else {
			sys.bcStack.PushF(c.ghv.fall_zvelocity * (c.localscl / oc.localscl))
		}
	case OC_ex_gethitvar_fall_recover:
		sys.bcStack.PushB(c.ghv.fall_recover)
	case OC_ex_gethitvar_fall_time:
		sys.bcStack.PushI(c.fallTime)
	case OC_ex_gethitvar_fall_recovertime:
		sys.bcStack.PushI(c.ghv.fall_recovertime)
	case OC_ex_gethitvar_fall_kill:
		sys.bcStack.PushB(c.ghv.fall_kill)
	case OC_ex_gethitvar_fall_envshake_time:
		sys.bcStack.PushI(c.ghv.fall_envshake_time)
	case OC_ex_gethitvar_fall_envshake_freq:
		sys.bcStack.PushF(c.ghv.fall_envshake_freq)
	case OC_ex_gethitvar_fall_envshake_ampl:
		sys.bcStack.PushI(int32(float32(c.ghv.fall_envshake_ampl) * (c.localscl / oc.localscl)))
	case OC_ex_gethitvar_fall_envshake_phase:
		sys.bcStack.PushF(c.ghv.fall_envshake_phase)
	case OC_ex_gethitvar_fall_envshake_mul:
		sys.bcStack.PushF(c.ghv.fall_envshake_mul)
	case OC_ex_gethitvar_attr:
		attr := (*(*int32)(unsafe.Pointer(&be[*i])))
		// same as c.hitDefAttr()
		sys.bcStack.PushB(c.ghv.testAttr(attr))
		*i += 4
	case OC_ex_gethitvar_dizzypoints:
		sys.bcStack.PushI(c.ghv.dizzypoints)
	case OC_ex_gethitvar_guardpoints:
		sys.bcStack.PushI(c.ghv.guardpoints)
	case OC_ex_gethitvar_id:
		sys.bcStack.PushI(c.ghv.playerId)
	case OC_ex_gethitvar_playerno:
		sys.bcStack.PushI(int32(c.ghv.playerNo) + 1)
	case OC_ex_gethitvar_redlife:
		sys.bcStack.PushI(c.ghv.redlife)
	case OC_ex_gethitvar_score:
		sys.bcStack.PushF(c.ghv.score)
	case OC_ex_gethitvar_hitdamage:
		sys.bcStack.PushI(c.ghv.hitdamage)
	case OC_ex_gethitvar_guarddamage:
		sys.bcStack.PushI(c.ghv.guarddamage)
	case OC_ex_gethitvar_power:
		sys.bcStack.PushI(c.ghv.power)
	case OC_ex_gethitvar_hitpower:
		sys.bcStack.PushI(c.ghv.hitpower)
	case OC_ex_gethitvar_guardpower:
		sys.bcStack.PushI(c.ghv.guardpower)
	case OC_ex_gethitvar_kill:
		sys.bcStack.PushB(c.ghv.kill)
	case OC_ex_gethitvar_priority:
		sys.bcStack.PushI(c.ghv.priority)
	case OC_ex_gethitvar_facing:
		sys.bcStack.PushI(c.ghv.facing)
	case OC_ex_gethitvar_ground_velocity_x:
		sys.bcStack.PushF(c.ghv.ground_velocity[0] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_ground_velocity_y:
		sys.bcStack.PushF(c.ghv.ground_velocity[1] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_ground_velocity_z:
		sys.bcStack.PushF(c.ghv.ground_velocity[2] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_air_velocity_x:
		sys.bcStack.PushF(c.ghv.air_velocity[0] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_air_velocity_y:
		sys.bcStack.PushF(c.ghv.air_velocity[1] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_air_velocity_z:
		sys.bcStack.PushF(c.ghv.air_velocity[2] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_down_velocity_x:
		sys.bcStack.PushF(c.ghv.down_velocity[0] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_down_velocity_y:
		sys.bcStack.PushF(c.ghv.down_velocity[1] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_down_velocity_z:
		sys.bcStack.PushF(c.ghv.down_velocity[2] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_guard_velocity_x:
		sys.bcStack.PushF(c.ghv.guard_velocity[0] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_guard_velocity_y:
		sys.bcStack.PushF(c.ghv.guard_velocity[1] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_guard_velocity_z:
		sys.bcStack.PushF(c.ghv.guard_velocity[2] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_airguard_velocity_x:
		sys.bcStack.PushF(c.ghv.airguard_velocity[0] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_airguard_velocity_y:
		sys.bcStack.PushF(c.ghv.airguard_velocity[1] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_airguard_velocity_z:
		sys.bcStack.PushF(c.ghv.airguard_velocity[2] * (c.localscl / oc.localscl))
	case OC_ex_gethitvar_frame:
		sys.bcStack.PushB(c.ghv.frame)
	case OC_ex_gethitvar_down_recover:
		sys.bcStack.PushB(c.ghv.down_recover)
	case OC_ex_gethitvar_guardflag:
		attr := (*(*int32)(unsafe.Pointer(&be[*i])))
		sys.bcStack.PushB(
			c.ghv.guardflag&attr != 0,
		)
		*i += 4
	case OC_ex_ailevelf:
		if !c.asf(ASF_noailevel) {
			sys.bcStack.PushF(c.getAILevel())
		} else {
			sys.bcStack.PushI(0)
		}
	case OC_ex_airjumpcount:
		sys.bcStack.PushI(c.airJumpCount)
	// AnimelemVar
	case OC_ex_animelemvar_alphadest, OC_ex_animelemvar_alphasource, OC_ex_animelemvar_angle,
		OC_ex_animelemvar_group, OC_ex_animelemvar_hflip, OC_ex_animelemvar_image,
		OC_ex_animelemvar_time, OC_ex_animelemvar_vflip, OC_ex_animelemvar_xoffset,
		OC_ex_animelemvar_xscale, OC_ex_animelemvar_yoffset, OC_ex_animelemvar_yscale,
		OC_ex_animelemvar_numclsn1, OC_ex_animelemvar_numclsn2:
		// Check for valid animation frame
		var f *AnimFrame
		if c.anim != nil {
			f = c.anim.CurrentFrame()
		}
		// Handle output
		if f != nil {
			switch opc {
			case OC_ex_animelemvar_alphadest:
				sys.bcStack.PushI(int32(f.DstAlpha))
			case OC_ex_animelemvar_alphasource:
				sys.bcStack.PushI(int32(f.SrcAlpha))
			case OC_ex_animelemvar_angle:
				sys.bcStack.PushF(f.Angle)
			case OC_ex_animelemvar_group:
				sys.bcStack.PushI(int32(f.Group))
			case OC_ex_animelemvar_hflip:
				sys.bcStack.PushB(f.Hscale < 0)
			case OC_ex_animelemvar_image:
				sys.bcStack.PushI(int32(f.Number))
			case OC_ex_animelemvar_time:
				sys.bcStack.PushI(f.Time)
			case OC_ex_animelemvar_vflip:
				sys.bcStack.PushB(f.Vscale < 0)
			case OC_ex_animelemvar_xoffset:
				sys.bcStack.PushI(int32(f.Xoffset))
			case OC_ex_animelemvar_xscale:
				sys.bcStack.PushF(f.Xscale)
			case OC_ex_animelemvar_yoffset:
				sys.bcStack.PushI(int32(f.Yoffset))
			case OC_ex_animelemvar_yscale:
				sys.bcStack.PushF(f.Yscale)
			case OC_ex_animelemvar_numclsn1:
				sys.bcStack.PushI(int32(len(f.Clsn1)))
			case OC_ex_animelemvar_numclsn2:
				sys.bcStack.PushI(int32(len(f.Clsn2)))
			}
		} else {
			sys.bcStack.Push(BytecodeSF())
		}
	case OC_ex_animlength:
		sys.bcStack.PushI(c.anim.totaltime)
	case OC_ex_animplayerno:
		sys.bcStack.PushI(int32(c.animPN) + 1)
	case OC_ex_attack:
		sys.bcStack.PushF(c.attackMul[0] * 100)
	case OC_ex_clsnoverlap:
		c2 := sys.bcStack.Pop().ToI()
		id := sys.bcStack.Pop().ToI()
		c1 := sys.bcStack.Pop().ToI()
		sys.bcStack.PushB(c.clsnOverlapTrigger(c1, id, c2))
	case OC_ex_combocount:
		sys.bcStack.PushI(c.comboCount())
	case OC_ex_consecutivewins:
		sys.bcStack.PushI(c.consecutiveWins())
	case OC_ex_decisiveround:
		sys.bcStack.PushB(sys.decisiveRound[^c.playerNo&1])
	case OC_ex_defence:
		sys.bcStack.PushF(float32(c.finalDefense * 100))
	case OC_ex_dizzy:
		sys.bcStack.PushB(c.scf(SCF_dizzy))
	case OC_ex_dizzypoints:
		sys.bcStack.PushI(c.dizzyPoints)
	case OC_ex_dizzypointsmax:
		sys.bcStack.PushI(c.dizzyPointsMax)
	case OC_ex_envshakevar_time:
		sys.bcStack.PushI(sys.envShake.time)
	case OC_ex_envshakevar_freq:
		sys.bcStack.PushF(sys.envShake.freq / float32(math.Pi) * 180)
	case OC_ex_envshakevar_ampl:
		sys.bcStack.PushF(float32(math.Abs(float64(sys.envShake.ampl / oc.localscl))))
	case OC_ex_fightscreenvar_info_author:
		sys.bcStack.PushB(sys.lifebar.authorLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_ex_fightscreenvar_info_localcoord_x:
		sys.bcStack.PushI(sys.lifebarLocalcoord[0])
	case OC_ex_fightscreenvar_info_localcoord_y:
		sys.bcStack.PushI(sys.lifebarLocalcoord[1])
	case OC_ex_fightscreenvar_info_name:
		sys.bcStack.PushB(sys.lifebar.nameLow ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_ex_fightscreenvar_round_ctrl_time:
		sys.bcStack.PushI(sys.lifebar.ro.ctrl_time)
	case OC_ex_fightscreenvar_round_over_hittime:
		sys.bcStack.PushI(sys.lifebar.ro.over_hittime)
	case OC_ex_fightscreenvar_round_over_time:
		sys.bcStack.PushI(sys.lifebar.ro.over_time)
	case OC_ex_fightscreenvar_round_over_waittime:
		sys.bcStack.PushI(sys.lifebar.ro.over_waittime)
	case OC_ex_fightscreenvar_round_over_wintime:
		sys.bcStack.PushI(sys.lifebar.ro.over_wintime)
	case OC_ex_fightscreenvar_round_slow_time:
		sys.bcStack.PushI(sys.lifebar.ro.slow_time)
	case OC_ex_fightscreenvar_round_start_waittime:
		sys.bcStack.PushI(sys.lifebar.ro.start_waittime)
	case OC_ex_fightscreenvar_round_callfight_time:
		sys.bcStack.PushI(sys.lifebar.ro.callfight_time)
	case OC_ex_fightscreenvar_time_framespercount:
		sys.bcStack.PushI(sys.lifebar.ti.framespercount)
	case OC_ex_fighttime:
		sys.bcStack.PushI(sys.gameTime)
	case OC_ex_firstattack:
		sys.bcStack.PushB(sys.firstAttack[c.teamside] == c.playerNo)
	case OC_ex_float:
		*sys.bcStack.Top() = BytecodeFloat(sys.bcStack.Top().ToF())
	case OC_ex_gamemode:
		sys.bcStack.PushB(strings.ToLower(sys.gameMode) ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_ex_groundangle:
		sys.bcStack.PushF(c.groundAngle)
	case OC_ex_groundlevel:
		sys.bcStack.PushF(c.groundLevel * (c.localscl / oc.localscl))
	case OC_ex_guardbreak:
		sys.bcStack.PushB(c.scf(SCF_guardbreak))
	case OC_ex_guardcount:
		sys.bcStack.PushI(c.guardCount)
	case OC_ex_guardpoints:
		sys.bcStack.PushI(c.guardPoints)
	case OC_ex_guardpointsmax:
		sys.bcStack.PushI(c.guardPointsMax)
	case OC_ex_helperid:
		sys.bcStack.PushI(c.helperId)
	case OC_ex_helpername:
		sys.bcStack.PushB(c.helperIndex != 0 && strings.ToLower(c.name) ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_ex_helperindexexist:
		*sys.bcStack.Top() = c.helperByIndexExist(*sys.bcStack.Top())
	case OC_ex_hitoverridden:
		sys.bcStack.PushB(c.hoIdx >= 0)
	case OC_ex_ikemenversion:
		sys.bcStack.PushF(c.gi().ikemenverF)
	case OC_ex_incustomanim:
		sys.bcStack.PushB(c.animPN != c.playerNo)
	case OC_ex_incustomstate:
		sys.bcStack.PushB(c.ss.sb.playerNo != c.playerNo)
	case OC_ex_indialogue:
		sys.bcStack.PushB(sys.dialogueFlg)
	// InputTime
	case OC_ex_inputtime_B, OC_ex_inputtime_D, OC_ex_inputtime_F, OC_ex_inputtime_U, OC_ex_inputtime_L, OC_ex_inputtime_R,
		OC_ex_inputtime_a, OC_ex_inputtime_b, OC_ex_inputtime_c, OC_ex_inputtime_x, OC_ex_inputtime_y, OC_ex_inputtime_z,
		OC_ex_inputtime_s, OC_ex_inputtime_d, OC_ex_inputtime_w, OC_ex_inputtime_m:
		// Check for valid inputs
		if c.keyctrl[0] && c.cmd != nil {
			switch opc {
			case OC_ex_inputtime_B:
				sys.bcStack.PushI(c.cmd[0].Buffer.Bb)
			case OC_ex_inputtime_D:
				sys.bcStack.PushI(c.cmd[0].Buffer.Db)
			case OC_ex_inputtime_F:
				sys.bcStack.PushI(c.cmd[0].Buffer.Fb)
			case OC_ex_inputtime_U:
				sys.bcStack.PushI(c.cmd[0].Buffer.Ub)
			case OC_ex_inputtime_L:
				sys.bcStack.PushI(c.cmd[0].Buffer.Lb)
			case OC_ex_inputtime_R:
				sys.bcStack.PushI(c.cmd[0].Buffer.Rb)
			case OC_ex_inputtime_a:
				sys.bcStack.PushI(c.cmd[0].Buffer.ab)
			case OC_ex_inputtime_b:
				sys.bcStack.PushI(c.cmd[0].Buffer.bb)
			case OC_ex_inputtime_c:
				sys.bcStack.PushI(c.cmd[0].Buffer.cb)
			case OC_ex_inputtime_x:
				sys.bcStack.PushI(c.cmd[0].Buffer.xb)
			case OC_ex_inputtime_y:
				sys.bcStack.PushI(c.cmd[0].Buffer.yb)
			case OC_ex_inputtime_z:
				sys.bcStack.PushI(c.cmd[0].Buffer.zb)
			case OC_ex_inputtime_s:
				sys.bcStack.PushI(c.cmd[0].Buffer.sb)
			case OC_ex_inputtime_d:
				sys.bcStack.PushI(c.cmd[0].Buffer.db)
			case OC_ex_inputtime_w:
				sys.bcStack.PushI(c.cmd[0].Buffer.wb)
			case OC_ex_inputtime_m:
				sys.bcStack.PushI(c.cmd[0].Buffer.mb)
			}
		} else {
			sys.bcStack.Push(BytecodeSF())
		}
	case OC_ex_isassertedchar:
		sys.bcStack.PushB(c.asf(AssertSpecialFlag((*(*int64)(unsafe.Pointer(&be[*i]))))))
		*i += 8
	case OC_ex_isassertedglobal:
		sys.bcStack.PushB(sys.gsf(GlobalSpecialFlag((*(*int32)(unsafe.Pointer(&be[*i]))))))
		*i += 4
	case OC_ex_ishost:
		sys.bcStack.PushB(c.isHost())
	case OC_ex_jugglepoints:
		*sys.bcStack.Top() = c.jugglePoints(*sys.bcStack.Top())
	case OC_ex_layerno:
		sys.bcStack.PushI(c.layerNo)
	case OC_ex_localcoord_x:
		sys.bcStack.PushF(sys.cgi[c.playerNo].localcoord[0])
	case OC_ex_localcoord_y:
		sys.bcStack.PushF(sys.cgi[c.playerNo].localcoord[1])
	case OC_ex_maparray:
		sys.bcStack.PushF(c.mapArray[sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))]])
		*i += 4
	case OC_ex_max:
		v2 := sys.bcStack.Pop()
		be.max(sys.bcStack.Top(), v2)
	case OC_ex_min:
		v2 := sys.bcStack.Pop()
		be.min(sys.bcStack.Top(), v2)
	case OC_ex_movehitvar_cornerpush:
		sys.bcStack.PushF(c.mhv.cornerpush)
	case OC_ex_movehitvar_frame:
		sys.bcStack.PushB(c.mhv.frame)
	case OC_ex_movehitvar_id:
		sys.bcStack.PushI(c.mhv.playerId)
	case OC_ex_movehitvar_overridden:
		sys.bcStack.PushB(c.mhv.overridden)
	case OC_ex_movehitvar_playerno:
		sys.bcStack.PushI(int32(c.mhv.playerNo))
	case OC_ex_movehitvar_spark_x:
		sys.bcStack.PushF(c.mhv.sparkxy[0] * (c.localscl / oc.localscl))
	case OC_ex_movehitvar_spark_y:
		sys.bcStack.PushF(c.mhv.sparkxy[1] * (c.localscl / oc.localscl))
	case OC_ex_movehitvar_uniqhit:
		sys.bcStack.PushI(c.mhv.uniqhit)
	case OC_ex_numplayer:
		sys.bcStack.PushI(c.numPlayer())
	case OC_ex_clamp:
		v3 := sys.bcStack.Pop()
		v2 := sys.bcStack.Pop()
		be.clamp(sys.bcStack.Top(), v2, v3)
	case OC_ex_atan2:
		v2 := sys.bcStack.Pop()
		be.atan2(sys.bcStack.Top(), v2)
	case OC_ex_sign:
		be.sign(sys.bcStack.Top())
	case OC_ex_rad:
		be.rad(sys.bcStack.Top())
	case OC_ex_deg:
		be.deg(sys.bcStack.Top())
	case OC_ex_lastplayerid:
		sys.bcStack.PushI(sys.nextCharId - 1)
	case OC_ex_lerp:
		v3 := sys.bcStack.Pop()
		v2 := sys.bcStack.Pop()
		be.lerp(sys.bcStack.Top(), v2, v3)
	case OC_ex_memberno:
		sys.bcStack.PushI(int32(c.memberNo) + 1)
	case OC_ex_movecountered:
		sys.bcStack.PushI(c.moveCountered())
	case OC_ex_mugenversion:
		sys.bcStack.PushF(c.mugenVersionF())
	case OC_ex_pausetime:
		sys.bcStack.PushI(c.pauseTimeTrigger())
	case OC_ex_physics:
		sys.bcStack.PushB(c.ss.physics == StateType(be[*i]))
		*i++
	case OC_ex_playerno:
		sys.bcStack.PushI(int32(c.playerNo) + 1)
	case OC_ex_playerindexexist:
		*sys.bcStack.Top() = sys.playerIndexExist(*sys.bcStack.Top())
	case OC_ex_playernoexist:
		*sys.bcStack.Top() = sys.playerNoExist(*sys.bcStack.Top())
	case OC_ex_randomrange:
		v2 := sys.bcStack.Pop()
		be.random(sys.bcStack.Top(), v2)
	case OC_ex_ratiolevel:
		sys.bcStack.PushI(c.ocd().ratioLevel)
	case OC_ex_receiveddamage:
		sys.bcStack.PushI(c.receivedDmg)
	case OC_ex_receivedhits:
		sys.bcStack.PushI(c.receivedHits)
	case OC_ex_redlife:
		sys.bcStack.PushI(c.redLife)
	case OC_ex_round:
		v2 := sys.bcStack.Pop()
		be.round(sys.bcStack.Top(), v2)
	case OC_ex_roundtime:
		sys.bcStack.PushI(int32(sys.tickCount))
	case OC_ex_score:
		sys.bcStack.PushF(c.score())
	case OC_ex_scoretotal:
		sys.bcStack.PushF(c.scoreTotal())
	case OC_ex_selfstatenoexist:
		*sys.bcStack.Top() = c.selfStatenoExist(*sys.bcStack.Top())
	case OC_ex_sprpriority:
		sys.bcStack.PushI(c.sprPriority)
	case OC_ex_stagebackedgedist:
		sys.bcStack.PushI(int32(c.stageBackEdgeDist() * (c.localscl / oc.localscl)))
	case OC_ex_stagefrontedgedist:
		sys.bcStack.PushI(int32(c.stageFrontEdgeDist() * (c.localscl / oc.localscl)))
	case OC_ex_stagetime:
		sys.bcStack.PushI(sys.stage.stageTime)
	case OC_ex_standby:
		sys.bcStack.PushB(c.scf(SCF_standby))
	case OC_ex_teamleader:
		sys.bcStack.PushI(int32(c.teamLeader()))
	case OC_ex_teamsize:
		sys.bcStack.PushI(c.teamSize())
	case OC_ex_timeelapsed:
		sys.bcStack.PushI(timeElapsed())
	case OC_ex_timeremaining:
		sys.bcStack.PushI(timeRemaining())
	case OC_ex_timetotal:
		sys.bcStack.PushI(timeTotal())
	case OC_ex_playercount:
		sys.bcStack.PushI(sys.playercount())
	case OC_ex_pos_z:
		sys.bcStack.PushF(c.pos[2] * (c.localscl / oc.localscl))
	case OC_ex_vel_z:
		sys.bcStack.PushF(c.vel[2] * (c.localscl / oc.localscl))
	case OC_ex_prevanim:
		sys.bcStack.PushI(c.prevAnimNo)
	case OC_ex_prevmovetype:
		sys.bcStack.PushB(c.ss.prevMoveType == MoveType(be[*i])<<15)
		*i++
	case OC_ex_prevstatetype:
		sys.bcStack.PushB(c.ss.prevStateType == StateType(be[*i]))
		*i++
	case OC_ex_reversaldefattr:
		sys.bcStack.PushB(c.reversalDefAttr(*(*int32)(unsafe.Pointer(&be[*i]))))
		*i += 4
	case OC_ex_angle:
		if c.csf(CSF_angledraw) {
			sys.bcStack.PushF(c.angle)
		} else {
			sys.bcStack.PushF(0)
		}
	case OC_ex_scale_x:
		if c.csf(CSF_angledraw) {
			sys.bcStack.PushF(c.angleDrawScale[0])
		} else {
			sys.bcStack.PushF(1)
		}
	case OC_ex_scale_y:
		if c.csf(CSF_angledraw) {
			sys.bcStack.PushF(c.angleDrawScale[1])
		} else {
			sys.bcStack.PushF(1)
		}
	case OC_ex_scale_z:
		sys.bcStack.PushF(c.zScale)
	case OC_ex_offset_x:
		sys.bcStack.PushF(c.offset[0]) // Already in local scale
	case OC_ex_offset_y:
		sys.bcStack.PushF(c.offset[1])
	case OC_ex_alpha_s:
		if c.csf(CSF_trans) {
			sys.bcStack.PushI(c.alpha[0])
		} else {
			sys.bcStack.PushI(255)
		}
	case OC_ex_alpha_d:
		if c.csf(CSF_trans) {
			sys.bcStack.PushI(c.alpha[1])
		} else {
			sys.bcStack.PushI(0)
		}
	case OC_ex_selfcommand:
		if c.cmd == nil {
			sys.bcStack.PushB(false)
		} else {
			cmd, ok := c.cmd[sys.workingState.playerNo].Names[sys.stringPool[sys.workingState.playerNo].List[*(*int32)(unsafe.Pointer(&be[*i]))]]
			ok = ok && c.command(sys.workingState.playerNo, cmd)
			sys.bcStack.PushB(ok)
		}
		*i += 4
	default:
		sys.errLog.Printf("%v\n", be[*i-1])
		c.panic()
	}
}

func (be BytecodeExp) run_ex2(c *Char, i *int, oc *Char) {
	(*i)++
	opc := be[*i-1]
	correctScale := false
	camOff := float32(0)
	camCorrected := false
	switch opc {
	case OC_ex2_index:
		sys.bcStack.PushI(c.index)
	case OC_ex2_runorder:
		sys.bcStack.PushI(c.runorder)
	case OC_ex2_palfxvar_time:
		sys.bcStack.PushI(c.palfxvar(0))
	case OC_ex2_palfxvar_addr:
		sys.bcStack.PushI(c.palfxvar(1))
	case OC_ex2_palfxvar_addg:
		sys.bcStack.PushI(c.palfxvar(2))
	case OC_ex2_palfxvar_addb:
		sys.bcStack.PushI(c.palfxvar(3))
	case OC_ex2_palfxvar_mulr:
		sys.bcStack.PushI(c.palfxvar(4))
	case OC_ex2_palfxvar_mulg:
		sys.bcStack.PushI(c.palfxvar(5))
	case OC_ex2_palfxvar_mulb:
		sys.bcStack.PushI(c.palfxvar(6))
	case OC_ex2_palfxvar_color:
		sys.bcStack.PushF(c.palfxvar2(1))
	case OC_ex2_palfxvar_hue:
		sys.bcStack.PushF(c.palfxvar2(2))
	case OC_ex2_palfxvar_invertall:
		sys.bcStack.PushI(c.palfxvar(-1))
	case OC_ex2_palfxvar_invertblend:
		sys.bcStack.PushI(c.palfxvar(-2))
	case OC_ex2_palfxvar_bg_time:
		sys.bcStack.PushI(sys.palfxvar(0, 1))
	case OC_ex2_palfxvar_bg_addr:
		sys.bcStack.PushI(sys.palfxvar(1, 1))
	case OC_ex2_palfxvar_bg_addg:
		sys.bcStack.PushI(sys.palfxvar(2, 1))
	case OC_ex2_palfxvar_bg_addb:
		sys.bcStack.PushI(sys.palfxvar(3, 1))
	case OC_ex2_palfxvar_bg_mulr:
		sys.bcStack.PushI(sys.palfxvar(4, 1))
	case OC_ex2_palfxvar_bg_mulg:
		sys.bcStack.PushI(sys.palfxvar(5, 1))
	case OC_ex2_palfxvar_bg_mulb:
		sys.bcStack.PushI(sys.palfxvar(6, 1))
	case OC_ex2_palfxvar_bg_color:
		sys.bcStack.PushF(sys.palfxvar2(1, 1))
	case OC_ex2_palfxvar_bg_hue:
		sys.bcStack.PushF(sys.palfxvar2(2, 1))
	case OC_ex2_palfxvar_bg_invertall:
		sys.bcStack.PushI(sys.palfxvar(-1, 1))
	case OC_ex2_palfxvar_all_time:
		sys.bcStack.PushI(sys.palfxvar(0, 2))
	case OC_ex2_palfxvar_all_addr:
		sys.bcStack.PushI(sys.palfxvar(1, 2))
	case OC_ex2_palfxvar_all_addg:
		sys.bcStack.PushI(sys.palfxvar(2, 2))
	case OC_ex2_palfxvar_all_addb:
		sys.bcStack.PushI(sys.palfxvar(3, 2))
	case OC_ex2_palfxvar_all_mulr:
		sys.bcStack.PushI(sys.palfxvar(4, 2))
	case OC_ex2_palfxvar_all_mulg:
		sys.bcStack.PushI(sys.palfxvar(5, 2))
	case OC_ex2_palfxvar_all_mulb:
		sys.bcStack.PushI(sys.palfxvar(6, 2))
	case OC_ex2_palfxvar_all_color:
		sys.bcStack.PushF(sys.palfxvar2(1, 2))
	case OC_ex2_palfxvar_all_hue:
		sys.bcStack.PushF(sys.palfxvar2(2, 2))
	case OC_ex2_palfxvar_all_invertall:
		sys.bcStack.PushI(sys.palfxvar(-1, 2))
	case OC_ex2_palfxvar_all_invertblend:
		sys.bcStack.PushI(sys.palfxvar(-2, 2))
	case OC_ex2_introstate:
		sys.bcStack.PushI(sys.introState())
	case OC_ex2_outrostate:
		sys.bcStack.PushI(sys.outroState())
	case OC_ex2_bgmvar_filename:
		sys.bcStack.PushB(sys.bgm.filename ==
			sys.stringPool[sys.workingState.playerNo].List[*(*int32)(
				unsafe.Pointer(&be[*i]))])
		*i += 4
	case OC_ex2_bgmvar_freqmul:
		sys.bcStack.PushF(sys.bgm.freqmul)
	case OC_ex2_bgmvar_length:
		if sys.bgm.streamer == nil {
			sys.bcStack.PushI(0)
		} else {
			sys.bcStack.PushI(int32(sys.bgm.streamer.Len()))
		}
	case OC_ex2_bgmvar_loop:
		sys.bcStack.PushI(int32(sys.bgm.loop))
	case OC_ex2_bgmvar_loopcount:
		if sys.bgm.volctrl != nil {
			if sl, ok := sys.bgm.volctrl.Streamer.(*StreamLooper); ok {
				sys.bcStack.PushI(int32(sl.loopcount))
			} else {
				sys.bcStack.PushI(0)
			}
		} else {
			sys.bcStack.PushI(0)
		}
	case OC_ex2_bgmvar_loopend:
		if sys.bgm.volctrl != nil {
			if sl, ok := sys.bgm.volctrl.Streamer.(*StreamLooper); ok {
				sys.bcStack.PushI(int32(sl.loopend))
			} else {
				sys.bcStack.PushI(0)
			}
		} else {
			sys.bcStack.PushI(0)
		}
	case OC_ex2_bgmvar_loopstart:
		if sys.bgm.volctrl != nil {
			if sl, ok := sys.bgm.volctrl.Streamer.(*StreamLooper); ok {
				sys.bcStack.PushI(int32(sl.loopstart))
			} else {
				sys.bcStack.PushI(0)
			}
		} else {
			sys.bcStack.PushI(0)
		}
	case OC_ex2_bgmvar_position:
		if sys.bgm.streamer == nil {
			sys.bcStack.PushI(0)
		} else {
			sys.bcStack.PushI(int32(sys.bgm.streamer.Position()))
		}
	case OC_ex2_bgmvar_startposition:
		sys.bcStack.PushI(int32(sys.bgm.startPos))
	case OC_ex2_bgmvar_volume:
		sys.bcStack.PushI(int32(sys.bgm.bgmVolume))
	case OC_ex2_clsnvar_left:
		idx := int(sys.bcStack.Pop().ToI())
		id := int(sys.bcStack.Pop().ToI())
		v := float32(math.NaN())
		switch id {
		case 3: // DON'T ASK WHY BUT 0 CAUSES ERRORS, 3 DOES NOT
			v = c.sizeBox[0]
		case 1:
			cf1 := c.anim.CurrentFrame().Clsn1
			if cf1 != nil && idx >= 0 && idx < len(cf1) {
				v = cf1[idx][0]
			}
		case 2:
			cf2 := c.anim.CurrentFrame().Clsn2
			if cf2 != nil && idx >= 0 && idx < len(cf2) {
				v = cf2[idx][0]
			}
		}
		sys.bcStack.PushF(v * (c.localscl / oc.localscl))
	case OC_ex2_clsnvar_top:
		idx := int(sys.bcStack.Pop().ToI())
		id := int(sys.bcStack.Pop().ToI())
		v := float32(math.NaN())
		switch id {
		case 3: // DON'T ASK WHY BUT 0 CAUSES ERRORS, 3 DOES NOT
			v = c.sizeBox[1]
		case 1:
			cf1 := c.anim.CurrentFrame().Clsn1
			if cf1 != nil && idx >= 0 && idx < len(cf1) {
				v = cf1[idx][1]
			}
		case 2:
			cf2 := c.anim.CurrentFrame().Clsn2
			if cf2 != nil && idx >= 0 && idx < len(cf2) {
				v = cf2[idx][1]
			}
		}
		sys.bcStack.PushF(v * (c.localscl / oc.localscl))
	case OC_ex2_clsnvar_right:
		idx := int(sys.bcStack.Pop().ToI())
		id := int(sys.bcStack.Pop().ToI())
		v := float32(math.NaN())
		switch id {
		case 3: // DON'T ASK WHY BUT 0 CAUSES ERRORS, 3 DOES NOT
			v = c.sizeBox[2]
		case 1:
			cf1 := c.anim.CurrentFrame().Clsn1
			if cf1 != nil && idx >= 0 && idx < len(cf1) {
				v = cf1[idx][2]
			}
		case 2:
			cf2 := c.anim.CurrentFrame().Clsn2
			if cf2 != nil && idx >= 0 && idx < len(cf2) {
				v = cf2[idx][2]
			}
		}
		sys.bcStack.PushF(v * (c.localscl / oc.localscl))
	case OC_ex2_clsnvar_bottom:
		idx := int(sys.bcStack.Pop().ToI())
		id := int(sys.bcStack.Pop().ToI())
		v := float32(math.NaN())
		switch id {
		case 3: // DON'T ASK WHY BUT 0 CAUSES ERRORS, 3 DOES NOT
			v = c.sizeBox[3]
		case 1:
			cf1 := c.anim.CurrentFrame().Clsn1
			if cf1 != nil && idx >= 0 && idx < len(cf1) {
				v = cf1[idx][3]
			}
		case 2:
			cf2 := c.anim.CurrentFrame().Clsn2
			if cf2 != nil && idx >= 0 && idx < len(cf2) {
				v = cf2[idx][3]
			}
		}
		sys.bcStack.PushF(v * (c.localscl / oc.localscl))
	case OC_ex2_debug_accel:
		sys.bcStack.PushF(sys.accel)
	case OC_ex2_debug_clsndisplay:
		sys.bcStack.PushB(sys.clsnDisplay)
	case OC_ex2_debug_debugdisplay:
		sys.bcStack.PushB(sys.debugDisplay)
	case OC_ex2_debug_lifebardisplay:
		sys.bcStack.PushB(sys.lifebarDisplay)
	case OC_ex2_debug_roundreset:
		sys.bcStack.PushB(sys.roundResetFlg)
	case OC_ex2_debug_wireframedisplay:
		sys.bcStack.PushB(sys.wireframeDisplay)
	case OC_ex2_drawpal_group:
		sys.bcStack.PushI(c.drawPal()[0])
	case OC_ex2_drawpal_index:
		sys.bcStack.PushI(c.drawPal()[1])
	// BEGIN FALLTHROUGH (explodvar)
	case OC_ex2_explodvar_vel_x:
		correctScale = true
		fallthrough
	case OC_ex2_explodvar_vel_y:
		correctScale = true
		fallthrough
	case OC_ex2_explodvar_accel_x:
		correctScale = true
		fallthrough
	case OC_ex2_explodvar_accel_y:
		correctScale = true
		fallthrough
	case OC_ex2_explodvar_friction_x:
		correctScale = true
		fallthrough
	case OC_ex2_explodvar_friction_y:
		correctScale = true
		fallthrough
	case OC_ex2_explodvar_anim:
		fallthrough
	case OC_ex2_explodvar_animelem:
		fallthrough
	case OC_ex2_explodvar_drawpal_group:
		fallthrough
	case OC_ex2_explodvar_drawpal_index:
		fallthrough
	case OC_ex2_explodvar_removetime:
		fallthrough
	case OC_ex2_explodvar_pausemovetime:
		fallthrough
	case OC_ex2_explodvar_sprpriority:
		fallthrough
	case OC_ex2_explodvar_layerno:
		fallthrough
	case OC_ex2_explodvar_id:
		fallthrough
	case OC_ex2_explodvar_bindtime:
		fallthrough
	case OC_ex2_explodvar_time:
		fallthrough
	case OC_ex2_explodvar_facing:
		fallthrough
	case OC_ex2_explodvar_scale_x:
		fallthrough
	case OC_ex2_explodvar_scale_y:
		fallthrough
	case OC_ex2_explodvar_angle:
		fallthrough
	case OC_ex2_explodvar_angle_x:
		fallthrough
	case OC_ex2_explodvar_angle_y:
		camCorrected = true // gotta do this
		fallthrough
	case OC_ex2_explodvar_xshear:
		fallthrough
		// END FALLTHROUGH (explodvar)
	case OC_ex2_explodvar_pos_x:
		if !camCorrected {
			camOff = sys.cam.Pos[0] / c.localscl
			camCorrected = true
			correctScale = true
		}
		fallthrough
	case OC_ex2_explodvar_pos_y:
		if !camCorrected {
			camCorrected = true
			correctScale = true
		}
		idx := sys.bcStack.Pop()
		id := sys.bcStack.Pop()
		v := c.explodVar(id, idx, opc)
		if correctScale {
			sys.bcStack.PushF(v.ToF()*(c.localscl/oc.localscl) - camOff)
		} else {
			sys.bcStack.Push(v)
		}
	case OC_ex2_explodvar_pos_z:
		if !camCorrected {
			camCorrected = true
			correctScale = true
		}
		idx := sys.bcStack.Pop()
		id := sys.bcStack.Pop()
		v := c.explodVar(id, idx, opc)
		if correctScale {
			sys.bcStack.PushF(v.ToF()*(c.localscl/oc.localscl) - camOff)
		} else {
			sys.bcStack.Push(v)
		}
	// BEGIN FALLTHROUGH (projvar)
	case OC_ex2_projvar_accel_x:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_accel_y:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_accel_z:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_vel_x:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_vel_y:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_vel_z:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_projstagebound:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_projedgebound:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_lowbound:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_highbound:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_remvelocity_x:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_remvelocity_y:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_remvelocity_z:
		correctScale = true
		fallthrough
	case OC_ex2_projvar_projremove:
		fallthrough
	case OC_ex2_projvar_projremovetime:
		fallthrough
	case OC_ex2_projvar_projshadow_r:
		fallthrough
	case OC_ex2_projvar_projshadow_g:
		fallthrough
	case OC_ex2_projvar_projshadow_b:
		fallthrough
	case OC_ex2_projvar_projmisstime:
		fallthrough
	case OC_ex2_projvar_projhits:
		fallthrough
	case OC_ex2_projvar_projhitsmax:
		fallthrough
	case OC_ex2_projvar_projpriority:
		fallthrough
	case OC_ex2_projvar_projhitanim:
		fallthrough
	case OC_ex2_projvar_projremanim:
		fallthrough
	case OC_ex2_projvar_projcancelanim:
		fallthrough
	case OC_ex2_projvar_velmul_x:
		fallthrough
	case OC_ex2_projvar_velmul_y:
		fallthrough
	case OC_ex2_projvar_velmul_z:
		fallthrough
	case OC_ex2_projvar_projscale_x:
		fallthrough
	case OC_ex2_projvar_projscale_y:
		fallthrough
	case OC_ex2_projvar_projangle:
		fallthrough
	case OC_ex2_projvar_projxshear:
		fallthrough
	case OC_ex2_projvar_projsprpriority:
		fallthrough
	case OC_ex2_projvar_projlayerno:
		fallthrough
	case OC_ex2_projvar_projanim:
		fallthrough
	case OC_ex2_projvar_animelem:
		fallthrough
	case OC_ex2_projvar_drawpal_group:
		fallthrough
	case OC_ex2_projvar_drawpal_index:
		fallthrough
	case OC_ex2_projvar_supermovetime:
		fallthrough
	case OC_ex2_projvar_projid:
		fallthrough
	case OC_ex2_projvar_teamside:
		fallthrough
	case OC_ex2_projvar_pausemovetime:
		fallthrough
	case OC_ex2_projvar_pos_x:
		fallthrough
	case OC_ex2_projvar_pos_y:
		fallthrough
	case OC_ex2_projvar_pos_z:
		fallthrough
	case OC_ex2_projvar_facing:
		fallthrough
	case OC_ex2_projvar_time:
		fallthrough
	case OC_ex2_projvar_guardflag:
		fallthrough
	case OC_ex2_projvar_hitflag:
		flg := sys.bcStack.Pop()
		idx := sys.bcStack.Pop()
		id := sys.bcStack.Pop()
		v := c.projVar(id, idx, flg, opc, oc)
		sys.bcStack.Push(v)
	// END FALLTHROUGH (projvar)
	// FightScreenState
	case OC_ex2_fightscreenstate_fightdisplay:
		sys.bcStack.PushB(sys.lifebar.ro.triggerFightDisplay)
	case OC_ex2_fightscreenstate_kodisplay:
		sys.bcStack.PushB(sys.lifebar.ro.triggerKODisplay)
	case OC_ex2_fightscreenstate_rounddisplay:
		sys.bcStack.PushB(sys.lifebar.ro.triggerRoundDisplay)
	case OC_ex2_fightscreenstate_windisplay:
		sys.bcStack.PushB(sys.lifebar.ro.triggerWinDisplay)
	// MotifState
	case OC_ex2_motifstate_continuescreen:
		sys.bcStack.PushB(sys.continueScreenFlg)
	case OC_ex2_motifstate_victoryscreen:
		sys.bcStack.PushB(sys.victoryScreenFlg)
	case OC_ex2_motifstate_winscreen:
		sys.bcStack.PushB(sys.winScreenFlg)
	// SystemVar
	case OC_ex2_systemvar_introtime:
		if sys.intro > 0 {
			sys.bcStack.PushI(sys.intro)
		} else {
			sys.bcStack.PushI(0)
		}
	case OC_ex2_systemvar_outrotime:
		if sys.intro < 0 {
			sys.bcStack.PushI(-sys.intro)
		} else {
			sys.bcStack.PushI(0)
		}
	case OC_ex2_systemvar_pausetime:
		sys.bcStack.PushI(sys.pausetime)
	case OC_ex2_systemvar_slowtime:
		sys.bcStack.PushI(sys.slowtimeTrigger)
	case OC_ex2_systemvar_superpausetime:
		sys.bcStack.PushI(sys.supertime)
	// HitDefVar
	case OC_ex2_hitdefvar_guardflag:
		attr := (*(*int32)(unsafe.Pointer(&be[*i])))
		sys.bcStack.PushB(
			c.hitdef.guardflag&attr != 0,
		)
		*i += 4
	case OC_ex2_hitdefvar_hitflag:
		attr := (*(*int32)(unsafe.Pointer(&be[*i])))
		sys.bcStack.PushB(
			c.hitdef.hitflag&attr != 0,
		)
		*i += 4
	case OC_ex2_hitdefvar_hitdamage:
		sys.bcStack.PushI(c.hitdef.hitdamage)
	case OC_ex2_hitdefvar_guarddamage:
		sys.bcStack.PushI(c.hitdef.guarddamage)
	case OC_ex2_hitdefvar_p1stateno:
		sys.bcStack.PushI(c.hitdef.p1stateno)
	case OC_ex2_hitdefvar_p2stateno:
		sys.bcStack.PushI(c.hitdef.p2stateno)
	case OC_ex2_hitdefvar_priority:
		sys.bcStack.PushI(c.hitdef.priority)
	case OC_ex2_hitdefvar_id:
		sys.bcStack.PushI(c.hitdef.id)
	case OC_ex2_hitdefvar_sparkno:
		sys.bcStack.PushI(c.hitdef.sparkno)
	case OC_ex2_hitdefvar_guard_sparkno:
		sys.bcStack.PushI(c.hitdef.guard_sparkno)
	case OC_ex2_hitdefvar_sparkx:
		sys.bcStack.PushF(c.hitdef.sparkxy[0] * (c.localscl / oc.localscl))
	case OC_ex2_hitdefvar_sparky:
		sys.bcStack.PushF(c.hitdef.sparkxy[1] * (c.localscl / oc.localscl))
	case OC_ex2_hitdefvar_pausetime:
		sys.bcStack.PushI(c.hitdef.pausetime)
	case OC_ex2_hitdefvar_guard_pausetime:
		sys.bcStack.PushI(c.hitdef.guard_pausetime)
	case OC_ex2_hitdefvar_shaketime:
		sys.bcStack.PushI(c.hitdef.shaketime)
	case OC_ex2_hitdefvar_guard_shaketime:
		sys.bcStack.PushI(c.hitdef.guard_shaketime)
	case OC_ex2_hitdefvar_hitsound_group:
		sys.bcStack.PushI(c.hitdef.hitsound[0])
	case OC_ex2_hitdefvar_hitsound_number:
		sys.bcStack.PushI(c.hitdef.hitsound[1])
	case OC_ex2_hitdefvar_guardsound_group:
		sys.bcStack.PushI(c.hitdef.guardsound[0])
	case OC_ex2_hitdefvar_guardsound_number:
		sys.bcStack.PushI(c.hitdef.guardsound[1])
	// HitByAttr
	case OC_ex2_hitbyattr:
		sys.bcStack.PushB(c.hitByAttrTrigger(*(*int32)(unsafe.Pointer(&be[*i]))))
		*i += 4
	// BEGIN FALLTHROUGH (soundvar)
	case OC_ex2_soundvar_group:
		fallthrough
	case OC_ex2_soundvar_number:
		fallthrough
	case OC_ex2_soundvar_freqmul:
		fallthrough
	case OC_ex2_soundvar_isplaying:
		fallthrough
	case OC_ex2_soundvar_length:
		fallthrough
	case OC_ex2_soundvar_loopcount:
		fallthrough
	case OC_ex2_soundvar_loopend:
		fallthrough
	case OC_ex2_soundvar_loopstart:
		fallthrough
	case OC_ex2_soundvar_pan:
		fallthrough
	case OC_ex2_soundvar_position:
		fallthrough
	case OC_ex2_soundvar_priority:
		fallthrough
	case OC_ex2_soundvar_startposition:
		fallthrough
	case OC_ex2_soundvar_volumescale:
		// END FALLTHROUGH (soundvar)
		// get the channel
		ch := sys.bcStack.Pop()
		sys.bcStack.Push(c.soundVar(ch, opc))
	case OC_ex2_botboundbodydist:
		sys.bcStack.PushF(c.botBoundBodyDist() * (c.localscl / oc.localscl))
	case OC_ex2_botbounddist:
		sys.bcStack.PushF(c.botBoundDist() * (c.localscl / oc.localscl))
	case OC_ex2_topboundbodydist:
		sys.bcStack.PushF(c.topBoundBodyDist() * (c.localscl / oc.localscl))
	case OC_ex2_topbounddist:
		sys.bcStack.PushF(c.topBoundDist() * (c.localscl / oc.localscl))
	// StageBGVar
	case OC_ex2_stagebgvar_actionno,
		OC_ex2_stagebgvar_delta_x, OC_ex2_stagebgvar_delta_y,
		OC_ex2_stagebgvar_id, OC_ex2_stagebgvar_layerno,
		OC_ex2_stagebgvar_pos_x, OC_ex2_stagebgvar_pos_y,
		OC_ex2_stagebgvar_start_x, OC_ex2_stagebgvar_start_y,
		OC_ex2_stagebgvar_tile_x, OC_ex2_stagebgvar_tile_y,
		OC_ex2_stagebgvar_velocity_x, OC_ex2_stagebgvar_velocity_y:
		// Common inputs
		idx := int(sys.bcStack.Pop().ToI())
		id := sys.bcStack.Pop().ToI()
		bg := oc.getStageBg(id, idx, true)
		// Handle output
		if bg != nil {
			switch opc {
			case OC_ex2_stagebgvar_actionno:
				sys.bcStack.PushI(bg.actionno)
			case OC_ex2_stagebgvar_delta_x:
				sys.bcStack.PushF(bg.delta[0])
			case OC_ex2_stagebgvar_delta_y:
				sys.bcStack.PushF(bg.delta[1])
			case OC_ex2_stagebgvar_id:
				sys.bcStack.PushI(bg.id)
			case OC_ex2_stagebgvar_layerno:
				sys.bcStack.PushI(bg.layerno)
			case OC_ex2_stagebgvar_pos_x:
				sys.bcStack.PushF(bg.bga.pos[0] * sys.stage.localscl / oc.localscl)
			case OC_ex2_stagebgvar_pos_y:
				sys.bcStack.PushF(bg.bga.pos[1] * sys.stage.localscl / oc.localscl)
			case OC_ex2_stagebgvar_start_x:
				sys.bcStack.PushF(bg.start[0] * sys.stage.localscl / oc.localscl)
			case OC_ex2_stagebgvar_start_y:
				sys.bcStack.PushF(bg.start[1] * sys.stage.localscl / oc.localscl)
			case OC_ex2_stagebgvar_tile_x:
				sys.bcStack.PushI(bg.anim.tile.xflag)
			case OC_ex2_stagebgvar_tile_y:
				sys.bcStack.PushI(bg.anim.tile.yflag)
			case OC_ex2_stagebgvar_velocity_x:
				sys.bcStack.PushF(bg.bga.vel[0] * sys.stage.localscl / oc.localscl)
			case OC_ex2_stagebgvar_velocity_y:
				sys.bcStack.PushF(bg.bga.vel[1] * sys.stage.localscl / oc.localscl)
			}
		} else {
			sys.bcStack.Push(BytecodeSF())
		}
	case OC_ex2_numstagebg:
		*sys.bcStack.Top() = c.numStageBG(*sys.bcStack.Top())
	case OC_ex2_envshakevar_dir:
		sys.bcStack.PushF(sys.envShake.dir / float32(math.Pi) * 180)
	case OC_ex2_gethitvar_fall_envshake_dir:
		sys.bcStack.PushF(c.ghv.fall_envshake_dir)
	default:
		sys.errLog.Printf("%v\n", be[*i-1])
		c.panic()
	}
}

func (be BytecodeExp) evalF(c *Char) float32 {
	return be.run(c).ToF()
}

func (be BytecodeExp) evalI(c *Char) int32 {
	return be.run(c).ToI()
}

func (be BytecodeExp) evalI64(c *Char) int64 {
	return be.run(c).ToI64()
}

func (be BytecodeExp) evalB(c *Char) bool {
	return be.run(c).ToB()
}

type StateController interface {
	Run(c *Char, ps []int32) (changeState bool)
}

type NullStateController struct{}

func (NullStateController) Run(_ *Char, _ []int32) bool {
	return false
}

var nullStateController NullStateController

type bytecodeFunction struct {
	numVars int32
	numRets int32
	numArgs int32
	ctrls   []StateController
}

func (bf bytecodeFunction) run(c *Char, ret []uint8) (changeState bool) {
	oldv, oldvslen := sys.bcVar, len(sys.bcVarStack)
	sys.bcVar = sys.bcVarStack.Alloc(int(bf.numVars))
	if len(sys.bcStack) != int(bf.numArgs) {
		c.panic()
	}
	copy(sys.bcVar, sys.bcStack)
	sys.bcStack.Clear()
	for _, sc := range bf.ctrls {
		switch sc.(type) {
		case StateBlock:
		default:
			if c.hitPause() {
				continue
			}
		}
		if sc.Run(c, nil) {
			changeState = true
			break
		}
	}
	if !changeState {
		if len(ret) > 0 {
			if len(ret) != int(bf.numRets) {
				c.panic()
			}
			for i, r := range ret {
				oldv[r] = sys.bcVar[int(bf.numArgs)+i]
			}
		}
	}
	sys.bcVar, sys.bcVarStack = oldv, sys.bcVarStack[:oldvslen]
	return
}

type callFunction struct {
	bytecodeFunction
	arg BytecodeExp
	ret []uint8
}

func (cf callFunction) Run(c *Char, _ []int32) (changeState bool) {
	if len(cf.arg) > 0 {
		sys.bcStack.Push(cf.arg.run(c))
	}
	return cf.run(c, cf.ret)
}

type StateBlock struct {
	// Basic block fields
	persistent          int32
	persistentIndex     int32
	ignorehitpause      int32
	ctrlsIgnorehitpause bool
	trigger             BytecodeExp
	elseBlock           *StateBlock
	ctrls               []StateController
	// Loop fields
	loopBlock        bool
	nestedInLoop     bool
	forLoop          bool
	forAssign        bool
	forCtrlVar       varAssign
	forExpression    [3]BytecodeExp
	forBegin, forEnd int32
	forIncrement     int32
}

func newStateBlock() *StateBlock {
	return &StateBlock{persistent: 1, persistentIndex: -1, ignorehitpause: -2}
}

func (b StateBlock) Run(c *Char, ps []int32) (changeState bool) {
	// Check if the character is currently in a hit pause
	if c.hitPause() {
		// If ignorehitpause is less than -1, do not proceed with this controller
		if b.ignorehitpause < -1 {
			return false
		}
		// If ignorehitpause is non-negative, use the hitPauseExecutionToggleFlags mechanism
		if b.ignorehitpause >= 0 {
			flag := &c.ss.hitPauseExecutionToggleFlags[sys.workingState.playerNo][b.ignorehitpause]
			// Toggle the flag
			*flag = !*flag
			// If the flag is now false, skip the execution of this controller during this tick
			if !*flag {
				return false
			}
		}
	}
	if b.persistentIndex >= 0 {
		ps[b.persistentIndex]--
		if ps[b.persistentIndex] > 0 {
			return false
		}
	}
	// https://github.com/ikemen-engine/Ikemen-GO/issues/963
	//sys.workingChar = c
	sys.workingChar = sys.chars[c.ss.sb.playerNo][0]
	if b.loopBlock {
		if b.forLoop {
			if b.forAssign {
				// Initial assign to control variable
				b.forCtrlVar.Run(c, ps)
				b.forBegin = sys.bcVar[b.forCtrlVar.vari].ToI()
			} else {
				b.forBegin = b.forExpression[0].evalI(c)
			}
			b.forEnd, b.forIncrement = b.forExpression[1].evalI(c), b.forExpression[2].evalI(c)
		}
		// Start loop
		loopCount := 0
		interrupt := false
		for {
			// Decide if while loop should be stopped
			if !b.forLoop {
				// While loop needs to eval conditional indefinitely until it returns false
				if len(b.trigger) > 0 && !b.trigger.evalB(c) {
					interrupt = true
				}
			}
			// Run state controllers
			if !interrupt {
				for _, sc := range b.ctrls {
					switch sc.(type) {
					case StateBlock:
					default:
						if !b.ctrlsIgnorehitpause && c.hitPause() {
							continue
						}
					}
					if sc.Run(c, ps) {
						if sys.loopBreak {
							sys.loopBreak = false
							interrupt = true
							break
						}
						if sys.loopContinue {
							sys.loopContinue = false
							break
						}
						return true
					}
				}
			}
			// Decide if for loop should be stopped
			if b.forLoop {
				// Update loop count
				if b.forAssign {
					b.forBegin = sys.bcVar[b.forCtrlVar.vari].ToI() + b.forIncrement
				} else {
					b.forBegin += b.forIncrement
				}
				if b.forIncrement > 0 {
					if b.forBegin > b.forEnd {
						interrupt = true
					}
				} else if b.forBegin < b.forEnd {
					interrupt = true
				}
				// Update control variable if loop should keep going
				if b.forAssign && !interrupt {
					sys.bcVar[b.forCtrlVar.vari].SetI(b.forBegin)
				}
			}
			if interrupt {
				break
			}
			// Safety check. Prevents a bad loop from freezing Ikemen
			loopCount++
			if loopCount >= MaxLoop {
				sys.printBytecodeError(fmt.Sprintf("loop automatically stopped after %v iterations", loopCount))
				break
			}
		}
	} else {
		if len(b.trigger) > 0 && !b.trigger.evalB(c) {
			if b.elseBlock != nil {
				return b.elseBlock.Run(c, ps)
			}
			return false
		}
		for _, sc := range b.ctrls {
			switch sc.(type) {
			case StateBlock:
			default:
				if !b.ctrlsIgnorehitpause && c.hitPause() {
					continue
				}
			}
			if sc.Run(c, ps) {
				return true
			}
		}
	}
	if b.persistentIndex >= 0 {
		ps[b.persistentIndex] = b.persistent
	}
	return false
}

type StateExpr BytecodeExp

func (se StateExpr) Run(c *Char, _ []int32) (changeState bool) {
	BytecodeExp(se).run(c)
	return false
}

type varAssign struct {
	vari uint8
	be   BytecodeExp
}

func (va varAssign) Run(c *Char, _ []int32) (changeState bool) {
	sys.bcVar[va.vari] = va.be.run(c)
	return false
}

type LoopBreak struct{}

func (lb LoopBreak) Run(c *Char, _ []int32) (stop bool) {
	sys.loopBreak = true
	return true
}

type LoopContinue struct{}

func (lc LoopContinue) Run(c *Char, _ []int32) (stop bool) {
	sys.loopContinue = true
	return true
}

type StateControllerBase []byte

func newStateControllerBase() *StateControllerBase {
	return (*StateControllerBase)(&[]byte{})
}

func (StateControllerBase) beToExp(be ...BytecodeExp) []BytecodeExp {
	return be
}

func (StateControllerBase) fToExp(f ...float32) (exp []BytecodeExp) {
	for _, v := range f {
		var be BytecodeExp
		be.appendValue(BytecodeFloat(v))
		exp = append(exp, be)
	}
	return
}

func (StateControllerBase) iToExp(i ...int32) (exp []BytecodeExp) {
	for _, v := range i {
		var be BytecodeExp
		be.appendValue(BytecodeInt(v))
		exp = append(exp, be)
	}
	return
}

func (StateControllerBase) i64ToExp(i ...int64) (exp []BytecodeExp) {
	for _, v := range i {
		var be BytecodeExp
		be.appendValue(BytecodeInt64(v))
		exp = append(exp, be)
	}
	return
}

func (StateControllerBase) bToExp(i bool) (exp []BytecodeExp) {
	var be BytecodeExp
	be.appendValue(BytecodeBool(i))
	exp = append(exp, be)
	return
}

func (scb *StateControllerBase) add(paramID byte, exp []BytecodeExp) {
	*scb = append(*scb, paramID, byte(len(exp)))
	for _, e := range exp {
		l := int32(len(e))
		*scb = append(*scb, (*(*[4]byte)(unsafe.Pointer(&l)))[:]...)
		*scb = append(*scb, *(*[]byte)(unsafe.Pointer(&e))...)
	}
}

func (scb StateControllerBase) run(c *Char,
	f func(byte, []BytecodeExp) bool) {
	for i := 0; i < len(scb); {
		id := scb[i]
		i++
		n := scb[i]
		i++
		if cap(sys.workBe) < int(n) {
			sys.workBe = make([]BytecodeExp, n)
		} else {
			sys.workBe = sys.workBe[:n]
		}
		for m := 0; m < int(n); m++ {
			l := *(*int32)(unsafe.Pointer(&scb[i]))
			i += 4
			sys.workBe[m] = (*(*BytecodeExp)(unsafe.Pointer(&scb)))[i : i+int(l)]
			i += int(l)
		}
		if !f(id, sys.workBe) {
			break
		}
	}
}

type stateDef StateControllerBase

const (
	stateDef_hitcountpersist byte = iota
	stateDef_movehitpersist
	stateDef_hitdefpersist
	stateDef_sprpriority
	stateDef_facep2
	stateDef_juggle
	stateDef_velset
	stateDef_anim
	stateDef_ctrl
	stateDef_poweradd
)

func (sc stateDef) Run(c *Char) {
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case stateDef_hitcountpersist:
			if !exp[0].evalB(c) {
				c.clearHitCount()
			}
		case stateDef_movehitpersist:
			if !exp[0].evalB(c) {
				c.clearMoveHit()
			}
		case stateDef_hitdefpersist:
			if !exp[0].evalB(c) {
				c.clearHitDef()
				// Reset AttackDist
				c.hitdef.guard_dist_x = [2]float32{c.size.attack.dist.width[0], c.size.attack.dist.width[1]}
				c.hitdef.guard_dist_y = [2]float32{c.size.attack.dist.height[0], c.size.attack.dist.height[1]}
				c.hitdef.guard_dist_z = [2]float32{c.size.attack.dist.depth[0], c.size.attack.dist.depth[1]}
			}
		case stateDef_sprpriority:
			c.sprPriority = exp[0].evalI(c)
			c.layerNo = 0 // Prevent char from being forgotten in a different layer
		case stateDef_facep2:
			if exp[0].evalB(c) && !c.asf(ASF_nofacep2) && c.shouldFaceP2() {
				c.setFacing(-c.facing)
			}
		case stateDef_juggle:
			c.juggle = exp[0].evalI(c)
		case stateDef_velset:
			c.vel[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				c.vel[1] = exp[1].evalF(c)
				if len(exp) > 2 {
					c.vel[2] = exp[2].evalF(c)
				}
			}
		case stateDef_anim:
			c.changeAnimEx(exp[1].evalI(c), c.playerNo, string(*(*[]byte)(unsafe.Pointer(&exp[0]))), false)
		case stateDef_ctrl:
			c.setCtrl(exp[0].evalB(c))
		case stateDef_poweradd:
			c.powerAdd(exp[0].evalI(c))
		}
		return true
	})
}

type hitBy StateControllerBase

const (
	hitBy_value byte = iota
	hitBy_value2
	hitBy_time
	hitBy_attr
	hitBy_slot
	hitBy_playerno
	hitBy_playerid
	hitBy_stack
	hitBy_redirectid
)

func (sc hitBy) Run(c *Char, _ []int32) bool {
	crun := c
	slot := int(-1)
	attr := int32(-1)
	time := int32(1)
	pno := int(-1)
	pid := int32(-1)
	stk := false
	old := false
	set := func(slot int, attr, time int32, pno int, pid int32, stk bool) {
		crun.hitby[slot].not = false
		crun.hitby[slot].time = time
		crun.hitby[slot].flag = attr
		crun.hitby[slot].playerno = pno - 1
		crun.hitby[slot].playerid = pid
		crun.hitby[slot].stack = stk
	}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitBy_time:
			time = exp[0].evalI(c)
		case hitBy_value:
			val := exp[0].evalI(c)
			set(0, val, time, -1, -1, false)
			old = true
		case hitBy_value2:
			val := exp[0].evalI(c)
			set(1, val, time, -1, -1, false) // This redundancy is because both values can be set simultaneously in Mugen
			old = true
		case hitBy_slot:
			slot = int(Max(0, exp[0].evalI(c)))
			if slot > 7 {
				slot = 0
			}
		case hitBy_attr:
			attr = exp[0].evalI(c)
		case hitBy_playerno:
			pno = int(exp[0].evalI(c))
		case hitBy_playerid:
			pid = exp[0].evalI(c)
		case hitBy_stack:
			stk = exp[0].evalB(c)
		case hitBy_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if !old && slot >= 0 && slot <= 7 {
		set(slot, attr, time, pno, pid, stk)
	}
	return false
}

type notHitBy hitBy

func (sc notHitBy) Run(c *Char, _ []int32) bool {
	crun := c
	slot := int(-1)
	attr := int32(-1)
	time := int32(1)
	pno := int(-1)
	pid := int32(-1)
	stk := false
	old := false
	set := func(slot int, attr, time int32, pno int, pid int32, stk bool) {
		crun.hitby[slot].not = true
		crun.hitby[slot].time = time
		crun.hitby[slot].flag = ^attr // Opposite
		crun.hitby[slot].playerno = pno - 1
		crun.hitby[slot].playerid = pid
		crun.hitby[slot].stack = stk
	}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitBy_time:
			time = exp[0].evalI(c)
		case hitBy_value:
			val := exp[0].evalI(c)
			set(0, val, time, -1, -1, false)
			old = true
		case hitBy_value2:
			val := exp[0].evalI(c)
			set(1, val, time, -1, -1, false)
			old = true
		case hitBy_slot:
			slot = int(Max(0, exp[0].evalI(c)))
			if slot > 7 {
				slot = 0
			}
		case hitBy_attr:
			attr = exp[0].evalI(c)
		case hitBy_playerno:
			pno = int(exp[0].evalI(c))
		case hitBy_playerid:
			pid = exp[0].evalI(c)
		case hitBy_stack:
			stk = exp[0].evalB(c)
		case hitBy_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if !old && slot >= 0 && slot <= 7 {
		set(slot, attr, time, pno, pid, stk)
	}
	return false
}

type assertSpecial StateControllerBase

const (
	assertSpecial_flag byte = iota
	assertSpecial_flag_g
	assertSpecial_noko
	assertSpecial_enabled
	assertSpecial_redirectid
)

func (sc assertSpecial) Run(c *Char, _ []int32) bool {
	crun := c
	enable := true
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case assertSpecial_enabled:
			enable = exp[0].evalB(c)
		case assertSpecial_flag:
			flag := AssertSpecialFlag(exp[0].evalI64(c))
			if enable {
				crun.setASF(flag)
			} else {
				crun.unsetASF(flag)
			}
		case assertSpecial_flag_g:
			flag := GlobalSpecialFlag(exp[0].evalI64(c))
			if enable {
				sys.setGSF(flag)
			} else {
				sys.unsetGSF(flag)
			}
		case assertSpecial_noko:
			// NoKO affects all characters in Mugen, so legacy chars do so as well
			if c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
				if enable {
					sys.setGSF(GlobalSpecialFlag(GSF_globalnoko))
				} else {
					sys.unsetGSF(GlobalSpecialFlag(GSF_globalnoko))
				}
			} else {
				if enable {
					crun.setASF(AssertSpecialFlag(ASF_noko))
				} else {
					crun.unsetASF(AssertSpecialFlag(ASF_noko))
				}
			}
		case assertSpecial_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type playSnd StateControllerBase

const (
	playSnd_value = iota
	playSnd_channel
	playSnd_lowpriority
	playSnd_pan
	playSnd_abspan
	playSnd_volume
	playSnd_volumescale
	playSnd_freqmul
	playSnd_loop
	playSnd_redirectid
	playSnd_priority
	playSnd_loopstart
	playSnd_loopend
	playSnd_startposition
	playSnd_loopcount
	playSnd_stopongethit
	playSnd_stoponchangestate
)

func (sc playSnd) Run(c *Char, _ []int32) bool {
	if sys.noSoundFlg {
		return false
	}
	crun := c
	f, lw, lp, stopgh, stopcs := "", false, false, false, false
	var g, n, ch, vo, pri, lc int32 = -1, 0, -1, 100, 0, 0
	var loopstart, loopend, startposition = 0, 0, 0
	var p, fr float32 = 0, 1
	x := &c.pos[0]
	ls := c.localscl
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case playSnd_value:
			f = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			g = exp[1].evalI(c)
			if len(exp) > 2 {
				n = exp[2].evalI(c)
			}
		case playSnd_channel:
			ch = exp[0].evalI(c)
			if ch == 0 {
				stopgh = true
			}
		case playSnd_lowpriority:
			lw = exp[0].evalB(c)
		case playSnd_pan:
			p = exp[0].evalF(c)
		case playSnd_abspan:
			x = nil
			ls = 1
			p = exp[0].evalF(c)
		case playSnd_volume:
			vo = vo + int32(float64(exp[0].evalI(c))*(25.0/64.0))
		case playSnd_volumescale:
			vo = exp[0].evalI(c)
		case playSnd_freqmul:
			fr = ClampF(exp[0].evalF(c), 0.01, 5)
		case playSnd_loop:
			lp = exp[0].evalB(c)
		case playSnd_priority:
			pri = exp[0].evalI(c)
		case playSnd_loopstart:
			loopstart = int(exp[0].evalI64(c))
		case playSnd_loopend:
			loopend = int(exp[0].evalI64(c))
		case playSnd_startposition:
			startposition = int(exp[0].evalI64(c))
		case playSnd_loopcount:
			lc = exp[0].evalI(c)
		case playSnd_stopongethit:
			stopgh = exp[0].evalB(c)
		case playSnd_stoponchangestate:
			stopcs = exp[0].evalB(c)
		case playSnd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				x = &crun.pos[0]
				ls = crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	// Read the loop parameter if loopcount not specified
	if lc == 0 {
		if lp {
			crun.playSound(f, lw, -1, g, n, ch, vo, p, fr, ls, x, true, pri, loopstart, loopend, startposition, stopgh, stopcs)
		} else {
			crun.playSound(f, lw, 0, g, n, ch, vo, p, fr, ls, x, true, pri, loopstart, loopend, startposition, stopgh, stopcs)
		}
		// Use the loopcount directly if it's been specified
	} else {
		crun.playSound(f, lw, lc, g, n, ch, vo, p, fr, ls, x, true, pri, loopstart, loopend, startposition, stopgh, stopcs)
	}
	return false
}

type changeState StateControllerBase

const (
	changeState_value byte = iota
	changeState_ctrl
	changeState_anim
	changeState_continue
	changeState_readplayerid
	changeState_redirectid
)

func (sc changeState) Run(c *Char, _ []int32) bool {
	crun := c
	var v, a, ctrl int32 = -1, -1, -1
	ffx := ""
	stop := true
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case changeState_value:
			v = exp[0].evalI(c)
		case changeState_ctrl:
			ctrl = exp[0].evalI(c)
		case changeState_anim:
			a = exp[1].evalI(c)
			ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case changeState_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				stop = false
				crun = rid
			} else {
				return false
			}
		case changeState_continue:
			stop = !exp[0].evalB(c)
		}
		return true
	})
	crun.changeState(v, a, ctrl, ffx)
	return stop
}

type selfState changeState

func (sc selfState) Run(c *Char, _ []int32) bool {
	crun := c
	var v, a, r, ctrl int32 = -1, -1, -1, -1
	ffx := ""
	stop := true
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case changeState_value:
			v = exp[0].evalI(c)
		case changeState_ctrl:
			ctrl = exp[0].evalI(c)
		case changeState_anim:
			a = exp[1].evalI(c)
			ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case changeState_readplayerid:
			if rpid := sys.playerID(exp[0].evalI(c)); rpid != nil {
				r = int32(rpid.playerNo)
			} else {
				return false
			}
		case changeState_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				stop = false
				crun = rid
			} else {
				return false
			}
		case changeState_continue:
			stop = !exp[0].evalB(c)
		}
		return true
	})
	crun.selfState(v, a, r, ctrl, ffx)
	return stop
}

type tagIn StateControllerBase

const (
	tagIn_stateno = iota
	tagIn_partnerstateno
	tagIn_self
	tagIn_partner
	tagIn_ctrl
	tagIn_partnerctrl
	tagIn_leader
	tagIn_redirectid
)

func (sc tagIn) Run(c *Char, _ []int32) bool {
	crun := c
	var tagSCF int32 = -1
	var partnerNo int32 = -1
	var partnerStateNo int32 = -1
	var partnerCtrlSetting int32 = -1
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case tagIn_stateno:
			sn := exp[0].evalI(c)
			if sn >= 0 {
				crun.changeState(sn, -1, -1, "")
				if tagSCF == -1 {
					tagSCF = 1
				}
			} else {
				return false
			}
		case tagIn_partnerstateno:
			if psn := exp[0].evalI(c); psn >= 0 {
				partnerStateNo = psn
			} else {
				return false
			}
		case tagIn_self:
			tagSCF = Btoi(exp[0].evalB(c))
		case tagIn_partner:
			pti := exp[0].evalI(c)
			if pti >= 0 {
				partnerNo = pti
			} else {
				return false
			}
		case tagIn_ctrl:
			ctrls := exp[0].evalB(c)
			crun.setCtrl(ctrls)
			if tagSCF == -1 {
				tagSCF = 1
			}
		case tagIn_partnerctrl:
			partnerCtrlSetting = Btoi(exp[0].evalB(c))
		case tagIn_leader:
			if crun.teamside != -1 {
				ld := int(exp[0].evalI(c)) - 1
				if ld&1 == crun.playerNo&1 && ld >= crun.teamside && ld <= int(sys.numSimul[crun.teamside])*2-^crun.teamside&1-1 {
					sys.teamLeader[crun.playerNo&1] = ld
				}
			}
		case tagIn_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	// Data adjustments
	if tagSCF == -1 && partnerNo == -1 {
		tagSCF = 1
	}
	if tagSCF == 1 {
		crun.unsetSCF(SCF_standby)
	}
	// Partner
	if partnerNo != -1 && crun.partnerTag(partnerNo) != nil {
		partner := crun.partnerTag(partnerNo)
		partner.unsetSCF(SCF_standby)
		if partnerStateNo >= 0 {
			partner.changeState(partnerStateNo, -1, -1, "")
		}
		if partnerCtrlSetting != -1 {
			if partnerCtrlSetting == 1 {
				partner.setCtrl(true)
			} else {
				partner.setCtrl(false)
			}
		}
	}
	return false
}

type tagOut StateControllerBase

const (
	tagOut_self = iota
	tagOut_partner
	tagOut_stateno
	tagOut_partnerstateno
	tagOut_redirectid
)

func (sc tagOut) Run(c *Char, _ []int32) bool {
	crun := c
	var tagSCF int32 = -1
	var partnerNo int32 = -1
	var partnerStateNo int32 = -1
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case tagOut_self:
			tagSCF = Btoi(exp[0].evalB(c))
		case tagOut_stateno:
			sn := exp[0].evalI(c)
			if sn >= 0 {
				crun.changeState(sn, -1, -1, "")
				if tagSCF == -1 {
					tagSCF = 1
				}
			} else {
				return false
			}
		case tagOut_partner:
			pti := exp[0].evalI(c)
			if pti >= 0 {
				partnerNo = pti
			} else {
				return false
			}
		case tagOut_partnerstateno:
			if psn := exp[0].evalI(c); psn >= 0 {
				partnerStateNo = psn
			} else {
				return false
			}
		case tagOut_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if tagSCF == -1 && partnerNo == -1 && partnerStateNo == -1 {
		tagSCF = 1
	}
	if tagSCF == 1 {
		crun.setSCF(SCF_standby)
		// sys.charList.p2enemyDelete(crun)
	}
	if partnerNo != -1 && crun.partnerTag(partnerNo) != nil {
		partner := crun.partnerTag(partnerNo)
		partner.setSCF(SCF_standby)
		if partnerStateNo >= 0 {
			partner.changeState(partnerStateNo, -1, -1, "")
		}
		// sys.charList.p2enemyDelete(partner)
	}
	return false
}

type destroySelf StateControllerBase

const (
	destroySelf_recursive = iota
	destroySelf_removeexplods
	destroySelf_removetexts
	destroySelf_redirectid
)

func (sc destroySelf) Run(c *Char, _ []int32) bool {
	crun := c
	rec, rem, rtx := false, false, false
	self := true
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case destroySelf_recursive:
			rec = exp[0].evalB(c)
		case destroySelf_removeexplods:
			rem = exp[0].evalB(c)
		case destroySelf_removetexts:
			rtx = exp[0].evalB(c)
		case destroySelf_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				self = rid.id == c.id
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return crun.destroySelf(rec, rem, rtx) && self
}

type changeAnim StateControllerBase

const (
	changeAnim_elem byte = iota
	changeAnim_value
	changeAnim_readplayerid
	changeAnim_redirectid
)

func (sc changeAnim) Run(c *Char, _ []int32) bool {
	crun := c
	var elem int32
	var rpid int = -1
	setelem := false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case changeAnim_elem:
			elem = exp[0].evalI(c)
			setelem = true
		case changeAnim_value:
			pn := crun.playerNo // Default to own player number
			if rpid != -1 {
				pn = rpid
			}
			crun.changeAnim(exp[1].evalI(c), pn, string(*(*[]byte)(unsafe.Pointer(&exp[0]))))
			if setelem {
				crun.setAnimElem(elem)
			}
		case changeAnim_readplayerid:
			if read := sys.playerID(exp[0].evalI(c)); read != nil {
				rpid = read.playerNo
			} else {
				return false
			}
		case changeAnim_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type changeAnim2 changeAnim

func (sc changeAnim2) Run(c *Char, _ []int32) bool {
	crun := c
	var elem int32
	var rpid int = -1
	setelem := false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case changeAnim_elem:
			elem = exp[0].evalI(c)
			setelem = true
		case changeAnim_value:
			pn := crun.ss.sb.playerNo // Default to state owner player number
			if rpid != -1 {
				pn = rpid
			}
			crun.changeAnim2(exp[1].evalI(c), pn, string(*(*[]byte)(unsafe.Pointer(&exp[0]))))
			if setelem {
				crun.setAnimElem(elem)
			}
		case changeAnim_readplayerid:
			if read := sys.playerID(exp[0].evalI(c)); read != nil {
				rpid = read.playerNo
			} else {
				return false
			}
		case changeAnim_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type helper StateControllerBase

const (
	helper_helpertype byte = iota
	helper_name
	helper_postype
	helper_ownpal
	helper_size_xscale
	helper_size_yscale
	helper_size_ground_back
	helper_size_ground_front
	helper_size_air_back
	helper_size_air_front
	helper_size_height_stand
	helper_size_height_crouch
	helper_size_height_air
	helper_size_height_down
	helper_size_proj_doscale
	helper_size_head_pos
	helper_size_mid_pos
	helper_size_shadowoffset
	helper_size_depth
	helper_size_weight
	helper_size_pushfactor
	helper_stateno
	helper_keyctrl
	helper_id
	helper_pos
	helper_facing
	helper_pausemovetime
	helper_supermovetime
	helper_redirectid
	helper_remappal
	helper_extendsmap
	helper_inheritjuggle
	helper_inheritchannels
	helper_immortal
	helper_kovelocity
	helper_preserve
	helper_standby
	helper_ownclsnscale
)

func (sc helper) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	var h *Char
	pt := PT_P1
	var f, st int32 = 1, 0
	var extmap bool
	var x, y, z float32 = 0, 0, 0
	rp := [...]int32{-1, 0}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if h == nil {
			if paramID == helper_redirectid {
				if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
					crun = rid
					redirscale = c.localscl / crun.localscl
					h = crun.newHelper()
				} else {
					return false
				}
			} else {
				h = c.newHelper()
			}
		}
		if h == nil {
			return false
		}
		switch paramID {
		case helper_helpertype:
			ht := exp[0].evalI(c)
			switch ht {
			case 1:
				h.playerFlag = true
			case 2:
				h.hprojectile = true // Currently unused
			}
		case helper_name:
			h.name = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case helper_postype:
			pt = PosType(exp[0].evalI(c))
		case helper_ownpal:
			h.ownpal = exp[0].evalB(c)
		case helper_size_xscale:
			h.size.xscale = exp[0].evalF(c)
		case helper_size_yscale:
			h.size.yscale = exp[0].evalF(c)
		case helper_size_ground_back:
			h.size.ground.back = exp[0].evalF(c)
		case helper_size_ground_front:
			h.size.ground.front = exp[0].evalF(c)
		case helper_size_air_back:
			h.size.air.back = exp[0].evalF(c)
		case helper_size_air_front:
			h.size.air.front = exp[0].evalF(c)
		case helper_size_height_stand:
			h.size.height.stand = exp[0].evalF(c)
		case helper_size_height_crouch:
			h.size.height.crouch = exp[0].evalF(c)
		case helper_size_height_air:
			h.size.height.air[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				h.size.height.air[1] = exp[1].evalF(c)
			}
		case helper_size_height_down:
			h.size.height.down = exp[0].evalF(c)
		case helper_size_proj_doscale:
			h.size.proj.doscale = exp[0].evalI(c)
		case helper_size_head_pos:
			h.size.head.pos[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				h.size.head.pos[1] = exp[1].evalF(c)
			}
		case helper_size_mid_pos:
			h.size.mid.pos[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				h.size.mid.pos[1] = exp[1].evalF(c)
			}
		case helper_size_shadowoffset:
			h.size.shadowoffset = exp[0].evalF(c)
		case helper_size_depth:
			h.size.depth[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				h.size.depth[1] = exp[1].evalF(c)
			}
		case helper_size_weight:
			h.size.weight = exp[0].evalI(c)
		case helper_size_pushfactor:
			h.size.pushfactor = exp[0].evalF(c)
		case helper_stateno:
			st = exp[0].evalI(c)
		case helper_keyctrl:
			for _, e := range exp {
				m := e.run(c).ToI()
				if m > 0 && m <= int32(len(h.keyctrl)) {
					h.keyctrl[m-1] = true
				}
			}
		case helper_id:
			h.helperId = exp[0].evalI(c)
		case helper_pos:
			x = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				y = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					z = exp[2].evalF(c) * redirscale
				}
			}
		case helper_facing:
			f = exp[0].evalI(c)
		case helper_pausemovetime:
			h.pauseMovetime = exp[0].evalI(c)
		case helper_supermovetime:
			h.superMovetime = exp[0].evalI(c)
		case helper_remappal:
			rp[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				rp[1] = exp[1].evalI(c)
			}
		case helper_extendsmap:
			extmap = exp[0].evalB(c)
		case helper_inheritjuggle:
			h.inheritJuggle = exp[0].evalI(c)
		case helper_inheritchannels:
			h.inheritChannels = exp[0].evalI(c)
		case helper_immortal:
			h.immortal = exp[0].evalB(c)
		case helper_kovelocity:
			h.kovelocity = exp[0].evalB(c)
		case helper_preserve:
			if exp[0].evalB(c) {
				h.preserve = sys.round
			}
		case helper_ownclsnscale:
			h.ownclsnscale = exp[0].evalB(c)
		case helper_standby:
			if exp[0].evalB(c) {
				h.setSCF(SCF_standby)
			} else {
				h.unsetSCF(SCF_standby)
			}
		}
		return true
	})
	if h == nil {
		return false
	}
	if crun.minus == -2 || crun.minus == -4 {
		h.localscl = (320 / crun.localcoord)
		h.localcoord = crun.localcoord
	} else {
		h.localscl = crun.localscl
		h.localcoord = crun.localcoord
	}
	crun.helperInit(h, st, pt, x, y, z, f, rp, extmap)
	return false
}

type ctrlSet StateControllerBase

const (
	ctrlSet_value byte = iota
	ctrlSet_redirectid
)

func (sc ctrlSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case ctrlSet_value:
			crun.setCtrl(exp[0].evalB(c))
		case ctrlSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type posSet StateControllerBase

const (
	posSet_x byte = iota
	posSet_y
	posSet_z
	posSet_redirectid
)

func (sc posSet) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case posSet_x:
			x := sys.cam.Pos[0]/crun.localscl + exp[0].evalF(c)*redirscale
			crun.setX(x)
			if crun.bindToId > 0 && !math.IsNaN(float64(crun.bindPos[0])) && sys.playerID(crun.bindToId) != nil {
				crun.bindPosAdd[0] = x
			}
		case posSet_y:
			y := exp[0].evalF(c)*redirscale + crun.groundLevel + crun.platformPosY
			crun.setY(y)
			if crun.bindToId > 0 && !math.IsNaN(float64(crun.bindPos[1])) && sys.playerID(crun.bindToId) != nil {
				crun.bindPosAdd[1] = y
			}
		case posSet_z:
			z := exp[0].evalF(c) * redirscale
			crun.setZ(z)
			if crun.bindToId > 0 && !math.IsNaN(float64(crun.bindPos[2])) && sys.playerID(crun.bindToId) != nil {
				crun.bindPosAdd[2] = z
			}
		case posSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type posAdd posSet

func (sc posAdd) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case posSet_x:
			x := exp[0].evalF(c) * redirscale
			crun.addX(x)
			if crun.bindToId > 0 && !math.IsNaN(float64(crun.bindPos[0])) && sys.playerID(crun.bindToId) != nil {
				crun.bindPosAdd[0] = x
			}
		case posSet_y:
			y := exp[0].evalF(c) * redirscale
			crun.addY(y)
			if crun.bindToId > 0 && !math.IsNaN(float64(crun.bindPos[1])) && sys.playerID(crun.bindToId) != nil {
				crun.bindPosAdd[1] = y
			}
		case posSet_z:
			z := exp[0].evalF(c) * redirscale
			crun.addZ(z)
			if crun.bindToId > 0 && !math.IsNaN(float64(crun.bindPos[0])) && sys.playerID(crun.bindToId) != nil {
				crun.bindPosAdd[0] = z
			}
		case posSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type velSet posSet

func (sc velSet) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case posSet_x:
			crun.vel[0] = exp[0].evalF(c) * redirscale
		case posSet_y:
			crun.vel[1] = exp[0].evalF(c) * redirscale
		case posSet_z:
			crun.vel[2] = exp[0].evalF(c) * redirscale
		case posSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type velAdd posSet

func (sc velAdd) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case posSet_x:
			crun.vel[0] += exp[0].evalF(c) * redirscale
		case posSet_y:
			crun.vel[1] += exp[0].evalF(c) * redirscale
		case posSet_z:
			crun.vel[2] += exp[0].evalF(c) * redirscale
		case posSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type velMul posSet

func (sc velMul) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case posSet_x:
			crun.vel[0] *= exp[0].evalF(c)
		case posSet_y:
			crun.vel[1] *= exp[0].evalF(c)
		case posSet_z:
			crun.vel[2] *= exp[0].evalF(c)
		case posSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type modifyShadow StateControllerBase

const (
	modifyShadow_color byte = iota
	modifyShadow_intensity
	modifyShadow_offset
	modifyShadow_window
	modifyShadow_xshear
	modifyShadow_yscale
	modifyShadow_redirectid
)

func (sc modifyShadow) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyShadow_color:
			r := Clamp(exp[0].evalI(c), 0, 255)
			g := Clamp(exp[1].evalI(c), 0, 255)
			b := Clamp(exp[2].evalI(c), 0, 255)
			crun.shadowColor = [3]int32{r, g, b}
		case modifyShadow_intensity:
			crun.shadowIntensity = Clamp(exp[0].evalI(c), 0, 255)
		case modifyShadow_offset:
			crun.shadowOffset[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				crun.shadowOffset[1] = exp[1].evalF(c) * redirscale
			}
		case modifyShadow_window:
			crun.shadowWindow = [4]float32{exp[0].evalF(c), exp[1].evalF(c), exp[2].evalF(c), exp[3].evalF(c)}
		case modifyShadow_xshear:
			crun.shadowXshear = exp[0].evalF(c)
		case modifyShadow_yscale:
			crun.shadowYscale = exp[0].evalF(c)
		case modifyShadow_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type modifyReflection StateControllerBase

const (
	modifyReflection_color byte = iota
	modifyReflection_intensity
	modifyReflection_offset
	modifyReflection_window
	modifyReflection_xshear
	modifyReflection_yscale
	modifyReflection_redirectid
)

func (sc modifyReflection) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyReflection_color:
			r := Clamp(exp[0].evalI(c), 0, 255)
			g := Clamp(exp[1].evalI(c), 0, 255)
			b := Clamp(exp[2].evalI(c), 0, 255)
			crun.reflectColor = [3]int32{r, g, b}
		case modifyReflection_intensity:
			crun.reflectIntensity = Clamp(exp[0].evalI(c), 0, 255)
		case modifyReflection_offset:
			crun.reflectOffset[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				crun.reflectOffset[1] = exp[1].evalF(c) * redirscale
			}
		case modifyReflection_window:
			crun.reflectWindow = [4]float32{exp[0].evalF(c), exp[1].evalF(c), exp[2].evalF(c), exp[3].evalF(c)}
		case modifyReflection_xshear:
			crun.reflectXshear = exp[0].evalF(c)
		case modifyReflection_yscale:
			crun.reflectYscale = exp[0].evalF(c)
		case modifyReflection_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type palFX StateControllerBase

const (
	palFX_time byte = iota
	palFX_color
	palFX_add
	palFX_mul
	palFX_sinadd
	palFX_sinmul
	palFX_sincolor
	palFX_sinhue
	palFX_invertall
	palFX_invertblend
	palFX_hue
	palFX_last = iota - 1
	palFX_redirectid
)

func (sc palFX) runSub(c *Char, pfd *PalFXDef, paramID byte, exp []BytecodeExp) bool {
	switch paramID {
	case palFX_time:
		pfd.time = exp[0].evalI(c)
	case palFX_color:
		pfd.color = exp[0].evalF(c) / 256
	case palFX_hue:
		pfd.hue = exp[0].evalF(c) / 256
	case palFX_add:
		pfd.add[0] = exp[0].evalI(c)
		pfd.add[1] = exp[1].evalI(c)
		pfd.add[2] = exp[2].evalI(c)
	case palFX_mul:
		pfd.mul[0] = exp[0].evalI(c)
		pfd.mul[1] = exp[1].evalI(c)
		pfd.mul[2] = exp[2].evalI(c)
	case palFX_sinadd:
		var side int32 = 1
		if len(exp) > 3 {
			if exp[3].evalI(c) < 0 {
				pfd.cycletime[0] = -exp[3].evalI(c)
				side = -1
			} else {
				pfd.cycletime[0] = exp[3].evalI(c)
			}
		}
		pfd.sinadd[0] = exp[0].evalI(c) * side
		pfd.sinadd[1] = exp[1].evalI(c) * side
		pfd.sinadd[2] = exp[2].evalI(c) * side
	case palFX_sinmul:
		var side int32 = 1
		if len(exp) > 3 {
			if exp[3].evalI(c) < 0 {
				pfd.cycletime[1] = -exp[3].evalI(c)
				side = -1
			} else {
				pfd.cycletime[1] = exp[3].evalI(c)
			}
		}
		pfd.sinmul[0] = exp[0].evalI(c) * side
		pfd.sinmul[1] = exp[1].evalI(c) * side
		pfd.sinmul[2] = exp[2].evalI(c) * side
	case palFX_sincolor:
		var side int32 = 1
		if len(exp) > 1 {
			if exp[1].evalI(c) < 0 {
				pfd.cycletime[2] = -exp[1].evalI(c)
				side = -1
			} else {
				pfd.cycletime[2] = exp[1].evalI(c)
			}
		}
		pfd.sincolor = exp[0].evalI(c) * side
	case palFX_sinhue:
		var side int32 = 1
		if len(exp) > 1 {
			if exp[1].evalI(c) < 0 {
				pfd.cycletime[3] = -exp[1].evalI(c)
				side = -1
			} else {
				pfd.cycletime[3] = exp[1].evalI(c)
			}
		}
		pfd.sinhue = exp[0].evalI(c) * side
	case palFX_invertall:
		pfd.invertall = exp[0].evalB(c)
	case palFX_invertblend:
		pfd.invertblend = Clamp(exp[0].evalI(c), -1, 2)
	default:
		return false
	}
	return true
}

func (sc palFX) Run(c *Char, _ []int32) bool {
	crun := c
	doOnce := false
	pf := newPalFX()
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if paramID == palFX_redirectid {
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		if !doOnce {
			if !crun.ownpal {
				return false
			}
			pf = crun.palfx
			if pf == nil {
				pf = newPalFX()
			}
			pf.clear2(true)
			// Mugen 1.1 behavior if invertblend param is omitted (Only if char mugenversion = 1.1)
			if c.stWgi().mugenver[0] == 1 && c.stWgi().mugenver[1] == 1 && c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
				pf.invertblend = -2
			}
			doOnce = true
		}
		sc.runSub(c, &pf.PalFXDef, paramID, exp)
		return true
	})
	return false
}

type allPalFX palFX

func (sc allPalFX) Run(c *Char, _ []int32) bool {
	sys.allPalFX.clear()
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		palFX(sc).runSub(c, &sys.allPalFX.PalFXDef, paramID, exp)
		// Forcing 1.1 kind behavior
		sys.allPalFX.invertblend = Clamp(sys.allPalFX.invertblend, 0, 1)
		return true
	})
	return false
}

type bgPalFX palFX

const (
	bgPalFX_id byte = iota + palFX_last + 1
	bgPalFX_index
)

func (sc bgPalFX) Run(c *Char, _ []int32) bool {
	bgid := int32(-1)
	bgidx := int(-1)
	var backgrounds []*backGround

	pfx := *newPalFXDef()
	pfx.invertblend = -2 // Forcing 1.1 behavior

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case bgPalFX_id:
			bgid = exp[0].evalI(c)
		case bgPalFX_index:
			bgidx = int(exp[0].evalI(c))
		default:
			// Parse PalFX parameters
			palFX(sc).runSub(c, &pfx, paramID, exp)
		}
		return true
	})
	// Apply BGPalFX
	if bgid < 0 && bgidx < 0 {
		// Apply to stage itself
		sys.bgPalFX.clear()
		sys.bgPalFX.PalFXDef = pfx
		sys.bgPalFX.invertblend = -3
	} else {
		// Apply to specific elements
		backgrounds = c.getMultipleStageBg(bgid, bgidx, false)
		if len(backgrounds) == 0 {
			return false
		}
		for _, bg := range backgrounds {
			bg.palfx.clear()
			bg.palfx.PalFXDef = pfx
			bg.palfx.invertblend = -3
		}
	}
	return false
}

type explod StateControllerBase

const (
	explod_anim byte = iota + palFX_last + 1
	explod_ownpal
	explod_remappal
	explod_id
	explod_facing
	explod_vfacing
	explod_pos
	explod_random
	explod_postype
	explod_velocity
	explod_friction
	explod_accel
	explod_scale
	explod_bindtime
	explod_removetime
	explod_supermove
	explod_supermovetime
	explod_pausemovetime
	explod_sprpriority
	explod_layerno
	explod_under
	explod_ontop
	explod_strictontop
	explod_shadow
	explod_removeongethit
	explod_removeonchangestate
	explod_trans
	explod_animelem
	explod_animfreeze
	explod_angle
	explod_yangle
	explod_xangle
	explod_xshear
	explod_projection
	explod_focallength
	explod_ignorehitpause
	explod_bindid
	explod_space
	explod_window
	explod_interpolate_time
	explod_interpolate_animelem
	explod_interpolate_pos
	explod_interpolate_scale
	explod_interpolate_angle
	explod_interpolate_alpha
	explod_interpolate_focallength
	explod_interpolate_pfx_mul
	explod_interpolate_pfx_add
	explod_interpolate_pfx_color
	explod_interpolate_pfx_hue
	explod_interpolation
	explod_redirectid
	explod_last = iota + palFX_last + 1 - 1
)

func (sc explod) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	var e *Explod
	var i int
	//e, i := crun.newExplod()
	rp := [...]int32{-1, 0}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if e == nil {
			if paramID == explod_redirectid {
				if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
					crun = rid
					redirscale = c.localscl / crun.localscl
					e, i = crun.newExplod()
					if e == nil {
						return false
					}
					e.id = 0
				} else {
					return false
				}
			} else {
				e, i = crun.newExplod()
				if e == nil {
					return false
				}
				e.id = 0
			}
			// Mugenversion 1.1 chars default postype to "None"
			if c.stWgi().mugenver[0] == 1 && c.stWgi().mugenver[1] == 1 {
				e.postype = PT_None
			}
		}
		switch paramID {
		case explod_anim:
			ffx := string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			if ffx != "" && ffx != "s" {
				e.ownpal = true
			}
			e.animNo = exp[1].evalI(c)
			e.anim = crun.getAnim(e.animNo, ffx, true)
		case explod_ownpal:
			e.ownpal = exp[0].evalB(c)
		case explod_remappal:
			rp[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				rp[1] = exp[1].evalI(c)
			}
		case explod_id:
			e.id = Max(0, exp[0].evalI(c))
		case explod_facing:
			if exp[0].evalI(c) < 0 {
				e.relativef = -1
			} else {
				e.relativef = 1
			}
		case explod_vfacing:
			if exp[0].evalI(c) < 0 {
				e.vfacing = -1
			} else {
				e.vfacing = 1
			}
		case explod_pos:
			e.relativePos[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				e.relativePos[1] = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					e.relativePos[2] = exp[2].evalF(c) * redirscale
				}
			}
		case explod_random:
			rndx := (exp[0].evalF(c) / 2) * redirscale
			e.relativePos[0] += RandF(-rndx, rndx)
			if len(exp) > 1 {
				rndy := (exp[1].evalF(c) / 2) * redirscale
				e.relativePos[1] += RandF(-rndy, rndy)
				if len(exp) > 2 {
					rndz := (exp[2].evalF(c) / 2) * redirscale
					e.relativePos[2] += RandF(-rndz, rndz)
				}
			}
		case explod_space:
			e.space = Space(exp[0].evalI(c))
		case explod_postype:
			e.postype = PosType(exp[0].evalI(c))
		case explod_velocity:
			e.velocity[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				e.velocity[1] = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					e.velocity[2] = exp[2].evalF(c) * redirscale
				}
			}
		case explod_friction:
			e.friction[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				e.friction[1] = exp[1].evalF(c)
				if len(exp) > 2 {
					e.friction[2] = exp[2].evalF(c)
				}
			}
		case explod_accel:
			e.accel[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				e.accel[1] = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					e.accel[2] = exp[2].evalF(c) * redirscale
				}
			}
		case explod_scale:
			e.scale[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				e.scale[1] = exp[1].evalF(c)
			}
		case explod_bindtime:
			e.bindtime = exp[0].evalI(c)
		case explod_removetime:
			e.removetime = exp[0].evalI(c)
		case explod_supermove:
			if exp[0].evalB(c) {
				e.supermovetime = -1
			} else {
				e.supermovetime = 0
			}
		case explod_supermovetime:
			e.supermovetime = exp[0].evalI(c)
			if e.supermovetime >= 0 {
				e.supermovetime = Max(e.supermovetime, e.supermovetime+1)
			}
		case explod_pausemovetime:
			e.pausemovetime = exp[0].evalI(c)
			if e.pausemovetime >= 0 {
				e.pausemovetime = Max(e.pausemovetime, e.pausemovetime+1)
			}
		case explod_sprpriority:
			e.sprpriority = exp[0].evalI(c)
		case explod_layerno:
			l := exp[0].evalI(c)
			if l > 0 {
				e.layerno = 1
			} else if l < 0 {
				e.layerno = -1
			} else {
				e.layerno = 0
			}
		case explod_ontop:
			if exp[0].evalB(c) {
				e.layerno = 1
			} else {
				e.layerno = 0
			}
		case explod_strictontop:
			if e.layerno > 0 {
				e.sprpriority = 0
			}
		case explod_under:
			e.under = exp[0].evalB(c)
		case explod_shadow:
			e.shadow[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				e.shadow[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					e.shadow[2] = exp[2].evalI(c)
				}
			}
		case explod_removeongethit:
			e.removeongethit = exp[0].evalB(c)
		case explod_removeonchangestate:
			e.removeonchangestate = exp[0].evalB(c)
		case explod_trans:
			e.alpha[0] = exp[0].evalI(c)
			e.alpha[1] = exp[1].evalI(c)
			sa, da := e.alpha[0], e.alpha[1]

			if len(exp) >= 3 {
				e.alpha[0] = Clamp(e.alpha[0], 0, 255)
				e.alpha[1] = Clamp(e.alpha[1], 0, 255)
				//if len(exp) >= 4 {
				//	e.alpha[1] = ^e.alpha[1]
				//} else if e.alpha[0] == 1 && e.alpha[1] == 255 {

				//Add
				e.blendmode = 1
				//Sub
				if sa == 1 && da == 255 {
					e.blendmode = 2
				} else if sa == -1 && da == 0 {
					e.blendmode = 0
				}
				if e.alpha[0] == 1 && e.alpha[1] == 255 {
					e.alpha[0] = 0
				}
			}
		case explod_animelem:
			animelem := exp[0].evalI(c)
			e.animelem = animelem
			if e.anim != nil {
				e.anim.Action() // This being in this place can cause a nil animation crash
			}
			e.setAnimElem()
		case explod_animfreeze:
			e.animfreeze = exp[0].evalB(c)
		case explod_angle:
			e.anglerot[0] = exp[0].evalF(c)
		case explod_yangle:
			e.anglerot[2] = exp[0].evalF(c)
		case explod_xangle:
			e.anglerot[1] = exp[0].evalF(c)
		case explod_xshear:
			e.xshear = exp[0].evalF(c)
		case explod_focallength:
			e.fLength = exp[0].evalF(c)
		case explod_ignorehitpause:
			e.ignorehitpause = exp[0].evalB(c)
		case explod_bindid:
			bId := exp[0].evalI(c)
			if bId == -1 {
				bId = crun.id
			}
			e.setBind(bId)
		case explod_projection:
			e.projection = Projection(exp[0].evalI(c))
		case explod_window:
			e.window = [4]float32{exp[0].evalF(c) * redirscale, exp[1].evalF(c) * redirscale, exp[2].evalF(c) * redirscale, exp[3].evalF(c) * redirscale}
		default:
			if c.stWgi().mugenver[0] == 1 && c.stWgi().mugenver[1] == 1 && c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
				e.palfxdef.invertblend = -2
			}
			palFX(sc).runSub(c, &e.palfxdef, paramID, exp)

			explod(sc).setInterpolation(c, e, paramID, exp, &e.palfxdef)

		}
		return true
	})
	if e == nil {
		return false
	}
	// In this scenario the explod scale is constant in Mugen
	//if c.minus == -2 || c.minus == -4 {
	//	e.localscl = (320 / crun.localcoord)
	//} else {
	e.localscl = crun.localscl
	e.setStartParams(&e.palfxdef)
	e.setPos(crun)
	crun.insertExplodEx(i, rp)
	return false
}

func (sc explod) setInterpolation(c *Char, e *Explod, paramID byte, exp []BytecodeExp, pfd *PalFXDef) bool {
	switch paramID {
	case explod_interpolate_time:
		e.interpolate_time[0] = exp[0].evalI(c)
		if e.interpolate_time[0] < 0 {
			e.interpolate_time[0] = e.removetime
		}
		e.interpolate_time[1] = e.interpolate_time[0]
		if e.interpolate_time[0] > 0 {
			e.resetInterpolation(pfd)
			e.interpolate = true
			if e.ownpal {
				pfd.interpolate = true
				pfd.itime = e.interpolate_time[0]
			}
		}
	case explod_interpolate_animelem:
		e.interpolate_animelem[1] = exp[0].evalI(c)
		e.interpolate_animelem[0] = e.animelem
		e.interpolate_animelem[2] = e.interpolate_animelem[1]
	case explod_interpolate_pos:
		e.interpolate_pos[3] = exp[0].evalF(c)
		if len(exp) > 1 {
			e.interpolate_pos[4] = exp[1].evalF(c)
			if len(exp) > 2 {
				e.interpolate_pos[5] = exp[2].evalF(c)
			}
		}
	case explod_interpolate_scale:
		e.interpolate_scale[2] = exp[0].evalF(c)
		if len(exp) > 1 {
			e.interpolate_scale[3] = exp[1].evalF(c)
		}
	case explod_interpolate_alpha:
		e.interpolate_alpha[2] = exp[0].evalI(c)
		e.interpolate_alpha[3] = exp[1].evalI(c)
		e.interpolate_alpha[2] = Clamp(e.interpolate_alpha[2], 0, 255)
		e.interpolate_alpha[3] = Clamp(e.interpolate_alpha[3], 0, 255)
	case explod_interpolate_angle:
		e.interpolate_angle[3] = exp[0].evalF(c)
		if len(exp) > 1 {
			e.interpolate_angle[4] = exp[1].evalF(c)
		}
		if len(exp) > 2 {
			e.interpolate_angle[5] = exp[2].evalF(c)
		}
	case explod_interpolate_focallength:
		e.interpolate_fLength[1] = exp[0].evalF(c)
	case explod_interpolate_pfx_mul:
		pfd.imul[0] = exp[0].evalI(c)
		if len(exp) > 1 {
			pfd.imul[1] = exp[1].evalI(c)
		}
		if len(exp) > 2 {
			pfd.imul[2] = exp[2].evalI(c)
		}
	case explod_interpolate_pfx_add:
		pfd.iadd[0] = exp[0].evalI(c)
		if len(exp) > 1 {
			pfd.iadd[1] = exp[1].evalI(c)
		}
		if len(exp) > 2 {
			pfd.iadd[2] = exp[2].evalI(c)
		}
	case explod_interpolate_pfx_color:
		pfd.icolor[0] = exp[0].evalF(c) / 256
	case explod_interpolate_pfx_hue:
		pfd.ihue[0] = exp[0].evalF(c) / 256
	default:
	}
	return true
}

type modifyExplod explod

const (
	modifyexplod_redirectid = iota + explod_last + 1
	modifyexplod_index
)

func (sc modifyExplod) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	eid := int32(-1)
	idx := int32(-1)
	var expls []*Explod
	rp := [...]int32{-1, 0}
	remap := false
	ptexists := false

	// Mugen chars can only modify some parameters after defining PosType
	// Ikemen chars don't have this restriction
	paramlock := func() bool {
		return c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 && !ptexists
	}

	eachExpl := func(f func(e *Explod)) {
		if idx < 0 {
			for _, e := range expls {
				if idx < 0 {
					f(e)
				}
			}
		} else if idx < int32(len(expls)) {
			f(expls[idx])
		}
	}

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case explod_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		case explod_remappal:
			rp[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				rp[1] = exp[1].evalI(c)
			}
			remap = true
		case explod_id:
			eid = exp[0].evalI(c)
		case modifyexplod_index:
			idx = exp[0].evalI(c)
		default:
			if len(expls) == 0 {
				expls = crun.getExplods(eid)
				if len(expls) == 0 {
					return false
				}
				eachExpl(func(e *Explod) {
					if e.ownpal && remap {
						crun.remapPal(e.palfx, [...]int32{1, 1}, rp)
					}
				})
			}
			switch paramID {
			case explod_postype:
				// In Mugen many explod parameters are defaulted when not being modified
				// What possibly happens in Mugen is that all parameters are read first then only applied if PosType is defined
				if paramlock() {
					eachExpl(func(e *Explod) {
						if e.facing*e.relativef >= 0 { // See below
							e.relativef = 1
						}
						e.offset = [3]float32{0, 0, 0}
						e.setX(e.offset[0])
						e.setY(e.offset[1])
						e.setZ(e.offset[2])
						e.relativePos = [3]float32{0, 0, 0}
						e.velocity = [3]float32{0, 0, 0}
						e.accel = [3]float32{0, 0, 0}
						e.bindId = -2
						if e.bindtime == 0 {
							e.bindtime = 1
						}
						e.space = Space_none
					})
				}
				// Flag PosType as found
				// From this point onward, Mugen chars can modify more parameters (Ikemen chars always could)
				ptexists = true
				// Update actual PosType
				pt := PosType(exp[0].evalI(c))
				eachExpl(func(e *Explod) {
					e.postype = pt
				})
			case explod_space:
				// For some reason Mugen also requires a PosType declaration to be able to modify space
				if !paramlock() {
					spc := Space(exp[0].evalI(c))
					eachExpl(func(e *Explod) {
						e.space = spc
					})
				}
			case explod_facing:
				if !paramlock() {
					rf := exp[0].evalF(c)
					eachExpl(func(e *Explod) {
						// There's a bug in Mugen 1.1 where an explod that is facing left can't be flipped
						// https://github.com/ikemen-engine/Ikemen-GO/issues/1252
						// Ikemen chars just work as supposed to
						if c.stWgi().ikemenver[0] != 0 || c.stWgi().ikemenver[1] != 0 || e.facing*e.relativef >= 0 {
							e.relativef = rf
						}
					})
				}
			case explod_vfacing:
				if !paramlock() {
					vf := exp[0].evalF(c)
					eachExpl(func(e *Explod) {
						// There's a bug in Mugen 1.1 where an explod that is upside down can't be flipped
						// Ikemen chars just work as supposed to
						if e.vfacing >= 0 || c.stWgi().ikemenver[0] != 0 || c.stWgi().ikemenver[1] != 0 {
							e.vfacing = vf
						}
					})
				}
			case explod_pos:
				if !paramlock() {
					pos := exp[0].evalF(c) * redirscale
					eachExpl(func(e *Explod) {
						e.relativePos[0] = pos
					})
					if len(exp) > 1 {
						pos := exp[1].evalF(c) * redirscale
						eachExpl(func(e *Explod) {
							e.relativePos[1] = pos
						})
						if len(exp) > 2 {
							pos := exp[2].evalF(c) * redirscale
							eachExpl(func(e *Explod) {
								e.relativePos[2] = pos
							})
						}
					}
				}
			case explod_random:
				if !paramlock() {
					rndx := (exp[0].evalF(c) / 2) * redirscale
					rndx = RandF(-rndx, rndx)
					eachExpl(func(e *Explod) {
						e.relativePos[0] += rndx
					})
					if len(exp) > 1 {
						rndy := (exp[1].evalF(c) / 2) * redirscale
						rndy = RandF(-rndy, rndy)
						eachExpl(func(e *Explod) {
							e.relativePos[1] += rndy
						})
						if len(exp) > 2 {
							rndz := (exp[2].evalF(c) / 2) * redirscale
							rndz = RandF(-rndz, rndz)
							eachExpl(func(e *Explod) {
								e.relativePos[2] += rndz
							})
						}
					}
				}
			case explod_velocity:
				if !paramlock() {
					vel := exp[0].evalF(c) * redirscale
					eachExpl(func(e *Explod) {
						e.velocity[0] = vel
					})
					if len(exp) > 1 {
						vel := exp[1].evalF(c) * redirscale
						eachExpl(func(e *Explod) {
							e.velocity[1] = vel
						})
						if len(exp) > 2 {
							vel := exp[2].evalF(c) * redirscale
							eachExpl(func(e *Explod) {
								e.velocity[2] = vel
							})
						}
					}
				}
			case explod_friction:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachExpl(func(e *Explod) {
					e.friction[0] = v1
					e.friction[1] = v2
					e.friction[2] = v3
				})
			case explod_accel:
				if !paramlock() {
					accel := exp[0].evalF(c) * redirscale
					eachExpl(func(e *Explod) {
						e.accel[0] = accel
					})
					if len(exp) > 1 {
						accel := exp[1].evalF(c) * redirscale
						eachExpl(func(e *Explod) {
							e.accel[1] = accel
						})
						if len(exp) > 2 {
							accel := exp[2].evalF(c) * redirscale
							eachExpl(func(e *Explod) {
								e.accel[2] = accel
							})
						}
					}
				}
			case explod_scale:
				x := exp[0].evalF(c)
				eachExpl(func(e *Explod) {
					e.scale[0] = x
				})
				if len(exp) > 1 {
					y := exp[1].evalF(c)
					eachExpl(func(e *Explod) {
						e.scale[1] = y
					})
				}
			case explod_bindtime:
				t := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					e.bindtime = t
					// Bindtime fix (update bindtime according to current explod time)
					if (crun.stWgi().ikemenver[0] != 0 || crun.stWgi().ikemenver[1] != 0) && t > 0 {
						e.bindtime = e.time + t
					}
					e.setX(e.pos[0])
					e.setY(e.pos[1])
					e.setZ(e.pos[2])
				})
			case explod_removetime:
				t := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					e.removetime = t
					// Removetime fix (update removetime according to current explod time)
					if (crun.stWgi().ikemenver[0] != 0 || crun.stWgi().ikemenver[1] != 0) && t > 0 {
						e.removetime = e.time + t
					}
				})
			case explod_supermove:
				if exp[0].evalB(c) {
					eachExpl(func(e *Explod) {
						e.supermovetime = -1
					})
				} else {
					eachExpl(func(e *Explod) {
						e.supermovetime = 0
					})
				}
			case explod_supermovetime:
				t := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					e.supermovetime = t
					// Supermovetime fix (update supermovetime according to current explod time)
					if (crun.stWgi().ikemenver[0] != 0 || crun.stWgi().ikemenver[1] != 0) && t > 0 {
						e.supermovetime = e.time + t
					}
				})
			case explod_pausemovetime:
				t := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					e.pausemovetime = t
					// Pausemovetime fix (update pausemovetime according to current explod time)
					if (crun.stWgi().ikemenver[0] != 0 || crun.stWgi().ikemenver[1] != 0) && t > 0 {
						e.pausemovetime = e.time + t
					}
				})
			case explod_sprpriority:
				t := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					e.sprpriority = t
				})
			case explod_layerno:
				l := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					if l > 0 {
						e.layerno = 1
					} else if l < 0 {
						e.layerno = -1
					} else {
						e.layerno = 0
					}
				})
			case explod_ontop:
				if exp[0].evalB(c) {
					eachExpl(func(e *Explod) {
						e.layerno = 1
					})
				} else {
					eachExpl(func(e *Explod) {
						e.layerno = 0
					})
				}
			case explod_strictontop:
				eachExpl(func(e *Explod) {
					if e.layerno > 0 {
						e.sprpriority = 0
					}
				})
			case explod_under:
				if exp[0].evalB(c) {
					eachExpl(func(e *Explod) {
						e.under = exp[0].evalB(c)
					})
				}
			case explod_shadow:
				r := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					e.shadow[0] = r
				})
				if len(exp) > 1 {
					g := exp[1].evalI(c)
					eachExpl(func(e *Explod) {
						e.shadow[1] = g
					})
					if len(exp) > 2 {
						b := exp[2].evalI(c)
						eachExpl(func(e *Explod) {
							e.shadow[2] = b
						})
					}
				}
			case explod_removeongethit:
				t := exp[0].evalB(c)
				eachExpl(func(e *Explod) {
					e.removeongethit = t
				})
			case explod_removeonchangestate:
				t := exp[0].evalB(c)
				eachExpl(func(e *Explod) {
					e.removeonchangestate = t
				})
			case explod_trans:
				s, d := exp[0].evalI(c), exp[1].evalI(c)
				blendmode := 0
				if len(exp) >= 3 {
					s, d = Clamp(s, 0, 255), Clamp(d, 0, 255)
					//if len(exp) >= 4 {
					//	d = ^d
					//} else if s == 1 && d == 255 {

					//Add
					blendmode = 1
					//Sub
					if s == 1 && d == 255 {
						blendmode = 2
					} else if s == -1 && d == 0 {
						blendmode = 0
					}

					if s == 1 && d == 255 {
						s = 0
					}

				}
				eachExpl(func(e *Explod) {
					e.alpha = [...]int32{s, d}
					e.blendmode = int32(blendmode)
				})
			case explod_anim:
				if c.stWgi().ikemenver[0] != 0 || c.stWgi().ikemenver[1] != 0 { // You could not modify this one in Mugen
					animNo := exp[1].evalI(c)
					anim := crun.getAnim(animNo, string(*(*[]byte)(unsafe.Pointer(&exp[0]))), true)
					eachExpl(func(e *Explod) {
						e.anim = anim
						e.animNo = animNo
					})
				}
			case explod_animelem:
				animelem := exp[0].evalI(c)
				eachExpl(func(e *Explod) {
					e.interpolate_animelem[1] = -1
					e.animelem = animelem
					if e.anim != nil {
						e.anim.Action() // This being in this place can cause a nil animation crash
					}
					e.setAnimElem()
				})
			case explod_animfreeze:
				animfreeze := exp[0].evalB(c)
				eachExpl(func(e *Explod) {
					e.animfreeze = animfreeze
				})
			case explod_angle:
				a := exp[0].evalF(c)
				eachExpl(func(e *Explod) {
					e.anglerot[0] = a
				})
			case explod_yangle:
				ya := exp[0].evalF(c)
				eachExpl(func(e *Explod) {
					e.anglerot[2] = ya
				})
			case explod_xangle:
				xa := exp[0].evalF(c)
				eachExpl(func(e *Explod) {
					e.anglerot[1] = xa
				})
			case explod_xshear:
				xs := exp[0].evalF(c)
				eachExpl(func(e *Explod) {
					e.xshear = xs
				})
			case explod_projection:
				eachExpl(func(e *Explod) {
					e.projection = Projection(exp[0].evalI(c))
				})
			case explod_focallength:
				eachExpl(func(e *Explod) {
					e.fLength = exp[0].evalF(c)
				})
			case explod_window:
				eachExpl(func(e *Explod) {
					e.window = [4]float32{exp[0].evalF(c) * redirscale, exp[1].evalF(c) * redirscale, exp[2].evalF(c) * redirscale, exp[3].evalF(c) * redirscale}
				})
			case explod_ignorehitpause:
				if c.stWgi().ikemenver[0] != 0 || c.stWgi().ikemenver[1] != 0 { // You could not modify this one in Mugen
					ihp := exp[0].evalB(c)
					eachExpl(func(e *Explod) {
						e.ignorehitpause = ihp
					})
				}
			case explod_bindid:
				bId := exp[0].evalI(c)
				if bId == -1 {
					bId = crun.id
				}
				eachExpl(func(e *Explod) {
					e.setBind(bId)
				})
			case explod_interpolation:
				if c.stWgi().ikemenver[0] != 0 || c.stWgi().ikemenver[1] != 0 {
					interpolation := exp[0].evalB(c)
					eachExpl(func(e *Explod) {
						if e.interpolate != interpolation && e.interpolate_time[0] > 0 {
							e.interpolate_animelem[0] = e.start_animelem
							e.interpolate_animelem[1] = e.interpolate_animelem[2]
							if e.ownpal {
								pfd := e.palfx
								pfd.interpolate = interpolation
								pfd.itime = e.interpolate_time[0]
							}
							e.interpolate_time[1] = e.interpolate_time[0]
							e.interpolate = interpolation
						}
					})
				}
			default:
				eachExpl(func(e *Explod) {
					if e.ownpal {
						palFX(sc).runSub(c, &e.palfx.PalFXDef, paramID, exp)
					}
				})
			}
		}
		return true
	})
	// Update relative positions if postype was updated
	if ptexists {
		eachExpl(func(e *Explod) {
			e.setPos(crun)
		})
	}
	return false
}

type gameMakeAnim StateControllerBase

const (
	gameMakeAnim_pos byte = iota
	gameMakeAnim_random
	gameMakeAnim_under
	gameMakeAnim_anim
	gameMakeAnim_redirectid
)

func (sc gameMakeAnim) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	var e *Explod
	var i int
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if e == nil {
			if paramID == gameMakeAnim_redirectid {
				if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
					crun = rid
					redirscale = c.localscl / crun.localscl
					e, i = crun.newExplod()
					if e == nil {
						return false
					}
					e.id = 0
				} else {
					return false
				}
			} else {
				e, i = crun.newExplod()
				if e == nil {
					return false
				}
				e.id = 0
			}
			e.layerno, e.sprpriority, e.ownpal = 1, math.MinInt32, true
		}
		switch paramID {
		case gameMakeAnim_pos:
			e.relativePos[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				e.relativePos[1] = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					e.relativePos[2] = exp[2].evalF(c) * redirscale
				}
			}
		case gameMakeAnim_random:
			rndx := (exp[0].evalF(c) / 2) * redirscale
			e.relativePos[0] += RandF(-rndx, rndx)
			if len(exp) > 1 {
				rndy := (exp[1].evalF(c) / 2) * redirscale
				e.relativePos[1] += RandF(-rndy, rndy)
				if len(exp) > 2 {
					rndz := (exp[2].evalF(c) / 2) * redirscale
					e.relativePos[2] += RandF(-rndz, rndz)
				}
			}
		case gameMakeAnim_under:
			if exp[0].evalB(c) {
				e.layerno = 0
			}
		case gameMakeAnim_anim: // Minor: Mugen uses anim 0 if nothing is specified
			e.anim = crun.getAnim(exp[1].evalI(c), string(*(*[]byte)(unsafe.Pointer(&exp[0]))), true)
		}
		return true
	})
	if e == nil {
		return false
	}
	e.relativePos[0] -= float32(crun.size.draw.offset[0])
	e.relativePos[1] -= float32(crun.size.draw.offset[1])
	e.setPos(crun)
	crun.insertExplod(i)
	return false
}

type afterImage palFX

const (
	afterImage_trans = iota + palFX_last + 1
	afterImage_time
	afterImage_length
	afterImage_timegap
	afterImage_framegap
	afterImage_palcolor
	afterImage_palhue
	afterImage_palinvertall
	afterImage_palinvertblend
	afterImage_palbright
	afterImage_palcontrast
	afterImage_palpostbright
	afterImage_paladd
	afterImage_palmul
	afterImage_ignorehitpause
	afterImage_last = iota + palFX_last + 1 - 1
	afterImage_redirectid
)

func (sc afterImage) runSub(c *Char, ai *AfterImage, paramID byte, exp []BytecodeExp) {
	switch paramID {
	case afterImage_trans:
		ai.alpha[0] = exp[0].evalI(c)
		ai.alpha[1] = exp[1].evalI(c)
		if len(exp) >= 3 {
			ai.alpha[0] = Clamp(ai.alpha[0], 0, 255)
			ai.alpha[1] = Clamp(ai.alpha[1], 0, 255)
			//if len(exp) >= 4 {
			//	ai.alpha[1] = ^ai.alpha[1]
			//} else if ai.alpha[0] == 1 && ai.alpha[1] == 255 {
			if ai.alpha[0] == 1 && ai.alpha[1] == 255 {
				ai.alpha[0] = 0
			}
		}
	case afterImage_time:
		ai.time = exp[0].evalI(c)
	case afterImage_length:
		ai.length = exp[0].evalI(c)
	case afterImage_timegap:
		ai.timegap = Max(1, exp[0].evalI(c))
	case afterImage_framegap:
		ai.framegap = exp[0].evalI(c)
	case afterImage_palcolor:
		ai.setPalColor(exp[0].evalI(c))
	case afterImage_palhue:
		ai.setPalHueShift(exp[0].evalI(c))
	case afterImage_palinvertall:
		ai.setPalInvertall(exp[0].evalB(c))
	case afterImage_palinvertblend:
		ai.setPalInvertblend(exp[0].evalI(c))
	case afterImage_palbright:
		ai.setPalBrightR(exp[0].evalI(c))
		if len(exp) > 1 {
			ai.setPalBrightG(exp[1].evalI(c))
			if len(exp) > 2 {
				ai.setPalBrightB(exp[2].evalI(c))
			}
		}
	case afterImage_palcontrast:
		ai.setPalContrastR(exp[0].evalI(c))
		if len(exp) > 1 {
			ai.setPalContrastG(exp[1].evalI(c))
			if len(exp) > 2 {
				ai.setPalContrastB(exp[2].evalI(c))
			}
		}
	case afterImage_palpostbright:
		ai.postbright[0] = exp[0].evalI(c)
		if len(exp) > 1 {
			ai.postbright[1] = exp[1].evalI(c)
			if len(exp) > 2 {
				ai.postbright[2] = exp[2].evalI(c)
			}
		}
	case afterImage_paladd:
		ai.add[0] = exp[0].evalI(c)
		if len(exp) > 1 {
			ai.add[1] = exp[1].evalI(c)
			if len(exp) > 2 {
				ai.add[2] = exp[2].evalI(c)
			}
		}
	case afterImage_palmul:
		ai.mul[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			ai.mul[1] = exp[1].evalF(c)
			if len(exp) > 2 {
				ai.mul[2] = exp[2].evalF(c)
			}
		}
	case afterImage_ignorehitpause:
		ai.ignorehitpause = exp[0].evalB(c)
	}
}

func (sc afterImage) Run(c *Char, _ []int32) bool {
	crun := c
	doOnce := false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if paramID == afterImage_redirectid {
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		if !doOnce {
			crun.aimg.clear()
			// Mugen 1.1 behavior if invertblend param is omitted (Only if char mugenversion = 1.1)
			if c.stWgi().mugenver[0] == 1 && c.stWgi().mugenver[1] == 1 && c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
				crun.aimg.palfx[0].invertblend = -2
			}
			crun.aimg.time = 1
			doOnce = true
		}
		sc.runSub(c, &crun.aimg, paramID, exp)
		return true
	})
	crun.aimg.setupPalFX()
	return false
}

type afterImageTime StateControllerBase

const (
	afterImageTime_time byte = iota
	afterImageTime_redirectid
)

func (sc afterImageTime) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if paramID == afterImageTime_redirectid {
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		if crun.aimg.timegap <= 0 {
			return false
		}
		switch paramID {
		case afterImageTime_time:
			time := exp[0].evalI(c)
			if time == 1 {
				time = 0
			}
			crun.aimg.time = time
			crun.aimg.timecount = 0
		}
		return true
	})
	return false
}

type hitDef afterImage

const (
	hitDef_attr = iota + afterImage_last + 1
	hitDef_guardflag
	hitDef_hitflag
	hitDef_ground_type
	hitDef_air_type
	hitDef_animtype
	hitDef_air_animtype
	hitDef_fall_animtype
	hitDef_affectteam
	hitDef_teamside
	hitDef_id
	hitDef_chainid
	hitDef_nochainid
	hitDef_kill
	hitDef_guard_kill
	hitDef_fall_kill
	hitDef_hitonce
	hitDef_air_juggle
	hitDef_getpower
	hitDef_damage
	hitDef_givepower
	hitDef_numhits
	hitDef_hitsound
	hitDef_hitsound_channel
	hitDef_guardsound
	hitDef_guardsound_channel
	hitDef_priority
	hitDef_p1stateno
	hitDef_p2stateno
	hitDef_p2getp1state
	hitDef_p1sprpriority
	hitDef_p2sprpriority
	hitDef_forcestand
	hitDef_forcecrouch
	hitDef_forcenofall
	hitDef_fall_damage
	hitDef_fall_xvelocity
	hitDef_fall_yvelocity
	hitDef_fall_zvelocity
	hitDef_fall_recover
	hitDef_fall_recovertime
	hitDef_sparkno
	hitDef_sparkangle
	hitDef_guard_sparkno
	hitDef_guard_sparkangle
	hitDef_sparkxy
	hitDef_down_hittime
	hitDef_p1facing
	hitDef_p1getp2facing
	hitDef_mindist
	hitDef_maxdist
	hitDef_snap
	hitDef_p2facing
	hitDef_air_hittime
	hitDef_fall
	hitDef_air_fall
	hitDef_air_cornerpush_veloff
	hitDef_down_bounce
	hitDef_down_velocity
	hitDef_down_cornerpush_veloff
	hitDef_ground_hittime
	hitDef_guard_hittime
	hitDef_guard_dist_x
	hitDef_guard_dist_y
	hitDef_guard_dist_z
	hitDef_pausetime
	hitDef_guard_pausetime
	hitDef_air_velocity
	hitDef_airguard_velocity
	hitDef_ground_slidetime
	hitDef_guard_slidetime
	hitDef_guard_ctrltime
	hitDef_airguard_ctrltime
	hitDef_ground_velocity_x
	hitDef_ground_velocity_y
	hitDef_ground_velocity_z
	hitDef_guard_velocity
	hitDef_ground_cornerpush_veloff
	hitDef_guard_cornerpush_veloff
	hitDef_airguard_cornerpush_veloff
	hitDef_xaccel
	hitDef_yaccel
	hitDef_zaccel
	hitDef_envshake_time
	hitDef_envshake_ampl
	hitDef_envshake_phase
	hitDef_envshake_freq
	hitDef_envshake_mul
	hitDef_envshake_dir
	hitDef_fall_envshake_time
	hitDef_fall_envshake_ampl
	hitDef_fall_envshake_phase
	hitDef_fall_envshake_freq
	hitDef_fall_envshake_mul
	hitDef_fall_envshake_dir
	hitDef_dizzypoints
	hitDef_guardpoints
	hitDef_redlife
	hitDef_score
	hitDef_p2clsncheck
	hitDef_p2clsnrequire
	hitDef_down_recover
	hitDef_down_recovertime
	hitDef_attack_depth
	hitDef_sparkscale
	hitDef_guard_sparkscale
	hitDef_last = iota + afterImage_last + 1 - 1
	hitDef_redirectid
)

// Additions to Hitdef should ideally also be done to GetHitVarSet and ModifyProjectile
func (sc hitDef) runSub(c *Char, hd *HitDef, paramID byte, exp []BytecodeExp) bool {
	switch paramID {
	case hitDef_attr:
		hd.attr = exp[0].evalI(c)
	case hitDef_guardflag:
		hd.guardflag = exp[0].evalI(c)
	case hitDef_hitflag:
		hd.hitflag = exp[0].evalI(c)
	case hitDef_ground_type:
		hd.ground_type = HitType(exp[0].evalI(c))
	case hitDef_air_type:
		hd.air_type = HitType(exp[0].evalI(c))
	case hitDef_animtype:
		hd.animtype = Reaction(exp[0].evalI(c))
	case hitDef_air_animtype:
		hd.air_animtype = Reaction(exp[0].evalI(c))
	case hitDef_fall_animtype:
		hd.fall_animtype = Reaction(exp[0].evalI(c))
	case hitDef_affectteam:
		hd.affectteam = exp[0].evalI(c)
	case hitDef_teamside:
		n := exp[0].evalI(c)
		if n > 2 {
			hd.teamside = 2
		} else if n < 0 {
			hd.teamside = 0
		} else {
			hd.teamside = int(n)
		}
	case hitDef_id:
		hd.id = Max(0, exp[0].evalI(c))
	case hitDef_chainid:
		hd.chainid = exp[0].evalI(c)
	case hitDef_nochainid:
		for i := 0; i < int(math.Min(8, float64(len(exp)))); i++ {
			hd.nochainid[i] = exp[i].evalI(c)
		}
	case hitDef_kill:
		hd.kill = exp[0].evalB(c)
	case hitDef_guard_kill:
		hd.guard_kill = exp[0].evalB(c)
	case hitDef_fall_kill:
		hd.fall_kill = exp[0].evalB(c)
	case hitDef_hitonce:
		hd.hitonce = Btoi(exp[0].evalB(c))
	case hitDef_air_juggle:
		hd.air_juggle = exp[0].evalI(c)
	case hitDef_getpower:
		hd.hitgetpower = Max(IErr+1, exp[0].evalI(c))
		if len(exp) > 1 {
			hd.guardgetpower = Max(IErr+1, exp[1].evalI(c))
		}
	case hitDef_damage:
		hd.hitdamage = exp[0].evalI(c)
		if len(exp) > 1 {
			hd.guarddamage = exp[1].evalI(c)
		}
	case hitDef_givepower:
		hd.hitgivepower = Max(IErr+1, exp[0].evalI(c))
		if len(exp) > 1 {
			hd.guardgivepower = Max(IErr+1, exp[1].evalI(c))
		}
	case hitDef_numhits:
		hd.numhits = exp[0].evalI(c)
	case hitDef_hitsound:
		hd.hitsound_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		hd.hitsound[0] = exp[1].evalI(c)
		if len(exp) > 2 {
			hd.hitsound[1] = exp[2].evalI(c)
		}
	case hitDef_hitsound_channel:
		hd.hitsound_channel = exp[0].evalI(c)
	case hitDef_guardsound:
		hd.guardsound_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		hd.guardsound[0] = exp[1].evalI(c)
		if len(exp) > 2 {
			hd.guardsound[1] = exp[2].evalI(c)
		}
	case hitDef_guardsound_channel:
		hd.guardsound_channel = exp[0].evalI(c)
	case hitDef_priority:
		hd.priority = exp[0].evalI(c)
		hd.prioritytype = TradeType(exp[1].evalI(c))
	case hitDef_p1stateno:
		hd.p1stateno = exp[0].evalI(c)
	case hitDef_p2stateno:
		hd.p2stateno = exp[0].evalI(c)
		hd.p2getp1state = true
	case hitDef_p2getp1state:
		hd.p2getp1state = exp[0].evalB(c)
	case hitDef_p1sprpriority:
		hd.p1sprpriority = exp[0].evalI(c)
	case hitDef_p2sprpriority:
		hd.p2sprpriority = exp[0].evalI(c)
	case hitDef_forcestand:
		hd.forcestand = Btoi(exp[0].evalB(c))
	case hitDef_forcecrouch:
		hd.forcecrouch = Btoi(exp[0].evalB(c))
	case hitDef_forcenofall:
		hd.forcenofall = exp[0].evalB(c)
	case hitDef_fall_damage:
		hd.fall_damage = exp[0].evalI(c)
	case hitDef_fall_xvelocity:
		hd.fall_xvelocity = exp[0].evalF(c)
	case hitDef_fall_yvelocity:
		hd.fall_yvelocity = exp[0].evalF(c)
	case hitDef_fall_zvelocity:
		hd.fall_zvelocity = exp[0].evalF(c)
	case hitDef_fall_recover:
		hd.fall_recover = exp[0].evalB(c)
	case hitDef_fall_recovertime:
		hd.fall_recovertime = exp[0].evalI(c)
	case hitDef_sparkno:
		hd.sparkno_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		hd.sparkno = exp[1].evalI(c)
	case hitDef_sparkangle:
		hd.sparkangle = exp[0].evalF(c)
	case hitDef_guard_sparkno:
		hd.guard_sparkno_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		hd.guard_sparkno = exp[1].evalI(c)
	case hitDef_guard_sparkangle:
		hd.guard_sparkangle = exp[0].evalF(c)
	case hitDef_sparkxy:
		hd.sparkxy[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.sparkxy[1] = exp[1].evalF(c)
		}
	case hitDef_down_hittime:
		hd.down_hittime = exp[0].evalI(c)
	case hitDef_p1facing:
		hd.p1facing = exp[0].evalI(c)
	case hitDef_p1getp2facing:
		hd.p1getp2facing = exp[0].evalI(c)
	case hitDef_mindist:
		hd.mindist[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.mindist[1] = exp[1].evalF(c)
			if len(exp) > 2 {
				hd.mindist[2] = exp[2].evalF(c)
			}
		}
	case hitDef_maxdist:
		hd.maxdist[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.maxdist[1] = exp[1].evalF(c)
			if len(exp) > 2 {
				hd.maxdist[2] = exp[2].evalF(c)
			}
		}
	case hitDef_snap:
		hd.snap[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.snap[1] = exp[1].evalF(c)
			if len(exp) > 2 {
				hd.snap[2] = exp[2].evalF(c)
				if len(exp) > 3 {
					hd.snaptime = exp[3].evalI(c)
				}
			}
		}
	case hitDef_p2facing:
		hd.p2facing = exp[0].evalI(c)
	case hitDef_air_hittime:
		hd.air_hittime = exp[0].evalI(c)
	case hitDef_fall:
		hd.ground_fall = exp[0].evalB(c)
		hd.air_fall = hd.ground_fall
	case hitDef_air_fall:
		hd.air_fall = exp[0].evalB(c)
	case hitDef_air_cornerpush_veloff:
		hd.air_cornerpush_veloff = exp[0].evalF(c)
	case hitDef_down_bounce:
		hd.down_bounce = exp[0].evalB(c)
	case hitDef_down_velocity:
		hd.down_velocity[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.down_velocity[1] = exp[1].evalF(c)
		}
		if len(exp) > 2 {
			hd.down_velocity[2] = exp[2].evalF(c)
		}
	case hitDef_down_cornerpush_veloff:
		hd.down_cornerpush_veloff = exp[0].evalF(c)
	case hitDef_ground_hittime:
		hd.ground_hittime = exp[0].evalI(c)
	case hitDef_guard_hittime:
		hd.guard_hittime = exp[0].evalI(c)
	case hitDef_guard_dist_x:
		var v1, v2 float32
		v1 = exp[0].evalF(c)
		if len(exp) > 1 {
			v2 = exp[1].evalF(c)
		}
		// Mugen ignores these if they're negative, rather than clamping them
		// Maybe that's what it does for all positive only parameters
		if v1 >= 0 {
			hd.guard_dist_x[0] = v1
		}
		if v2 >= 0 {
			hd.guard_dist_x[1] = v2
		}
	case hitDef_guard_dist_y:
		var v1, v2 float32
		v1 = exp[0].evalF(c)
		if len(exp) > 1 {
			v2 = exp[1].evalF(c)
		}
		if v1 >= 0 {
			hd.guard_dist_y[0] = v1
		}
		if v2 >= 0 {
			hd.guard_dist_y[1] = v2
		}
	case hitDef_guard_dist_z:
		var v1, v2 float32
		v1 = exp[0].evalF(c)
		if len(exp) > 1 {
			v2 = exp[1].evalF(c)
		}
		if v1 >= 0 {
			hd.guard_dist_z[0] = v1
		}
		if v2 >= 0 {
			hd.guard_dist_z[1] = v2
		}
	case hitDef_pausetime:
		hd.pausetime = exp[0].evalI(c)
		if len(exp) > 1 {
			hd.shaketime = exp[1].evalI(c)
		}
	case hitDef_guard_pausetime:
		hd.guard_pausetime = exp[0].evalI(c)
		if len(exp) > 1 {
			hd.guard_shaketime = exp[1].evalI(c)
		}
	case hitDef_air_velocity:
		hd.air_velocity[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.air_velocity[1] = exp[1].evalF(c)
		}
		if len(exp) > 2 {
			hd.air_velocity[2] = exp[2].evalF(c)
		}
	case hitDef_airguard_velocity:
		hd.airguard_velocity[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.airguard_velocity[1] = exp[1].evalF(c)
		}
		if len(exp) > 2 {
			hd.airguard_velocity[2] = exp[2].evalF(c)
		}
	case hitDef_ground_slidetime:
		hd.ground_slidetime = exp[0].evalI(c)
	case hitDef_guard_slidetime:
		hd.guard_slidetime = exp[0].evalI(c)
	case hitDef_guard_ctrltime:
		hd.guard_ctrltime = exp[0].evalI(c)
	case hitDef_airguard_ctrltime:
		hd.airguard_ctrltime = exp[0].evalI(c)
	case hitDef_ground_velocity_x:
		hd.ground_velocity[0] = exp[0].evalF(c)
	case hitDef_ground_velocity_y:
		hd.ground_velocity[1] = exp[0].evalF(c)
	case hitDef_ground_velocity_z:
		hd.ground_velocity[2] = exp[0].evalF(c)
	case hitDef_guard_velocity:
		hd.guard_velocity[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.guard_velocity[1] = exp[1].evalF(c)
		}
		if len(exp) > 2 {
			hd.guard_velocity[2] = exp[2].evalF(c)
		}
	case hitDef_ground_cornerpush_veloff:
		hd.ground_cornerpush_veloff = exp[0].evalF(c)
	case hitDef_guard_cornerpush_veloff:
		hd.guard_cornerpush_veloff = exp[0].evalF(c)
	case hitDef_airguard_cornerpush_veloff:
		hd.airguard_cornerpush_veloff = exp[0].evalF(c)
	case hitDef_xaccel:
		hd.xaccel = exp[0].evalF(c)
	case hitDef_yaccel:
		hd.yaccel = exp[0].evalF(c)
	case hitDef_zaccel:
		hd.zaccel = exp[0].evalF(c)
	case hitDef_envshake_time:
		hd.envshake_time = exp[0].evalI(c)
	case hitDef_envshake_ampl:
		hd.envshake_ampl = exp[0].evalI(c)
	case hitDef_envshake_freq:
		hd.envshake_freq = MaxF(0, exp[0].evalF(c))
	case hitDef_envshake_phase:
		hd.envshake_phase = exp[0].evalF(c)
	case hitDef_envshake_mul:
		hd.envshake_mul = exp[0].evalF(c)
	case hitDef_envshake_dir:
		hd.envshake_dir = exp[0].evalF(c)
	case hitDef_fall_envshake_time:
		hd.fall_envshake_time = exp[0].evalI(c)
	case hitDef_fall_envshake_ampl:
		hd.fall_envshake_ampl = exp[0].evalI(c)
	case hitDef_fall_envshake_freq:
		hd.fall_envshake_freq = MaxF(0, exp[0].evalF(c))
	case hitDef_fall_envshake_phase:
		hd.fall_envshake_phase = exp[0].evalF(c)
	case hitDef_fall_envshake_mul:
		hd.fall_envshake_mul = exp[0].evalF(c)
	case hitDef_fall_envshake_dir:
		hd.fall_envshake_dir = exp[0].evalF(c)
	case hitDef_dizzypoints:
		hd.dizzypoints = Max(IErr+1, exp[0].evalI(c))
	case hitDef_guardpoints:
		hd.guardpoints = Max(IErr+1, exp[0].evalI(c))
	case hitDef_redlife:
		hd.hitredlife = Max(IErr+1, exp[0].evalI(c))
		if len(exp) > 1 {
			hd.guardredlife = exp[1].evalI(c)
		}
	case hitDef_score:
		hd.score[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.score[1] = exp[1].evalF(c)
		}
	case hitDef_p2clsncheck:
		v := exp[0].evalI(c)
		if v == 0 || v == 1 || v == 2 || v == 3 {
			hd.p2clsncheck = v
		} else {
			hd.p2clsncheck = -1
		}
	case hitDef_p2clsnrequire:
		v := exp[0].evalI(c)
		if v == 1 || v == 2 || v == 3 {
			hd.p2clsnrequire = v
		} else {
			hd.p2clsnrequire = 0
		}
	case hitDef_down_recover:
		hd.down_recover = exp[0].evalB(c)
	case hitDef_down_recovertime:
		hd.down_recovertime = exp[0].evalI(c)
	case hitDef_attack_depth:
		hd.attack_depth[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.attack_depth[1] = exp[1].evalF(c)
		} else {
			hd.attack_depth[1] = hd.attack_depth[0]
		}
	case hitDef_sparkscale:
		hd.sparkscale[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.sparkscale[1] = exp[1].evalF(c)
		}
	case hitDef_guard_sparkscale:
		hd.guard_sparkscale[0] = exp[0].evalF(c)
		if len(exp) > 1 {
			hd.guard_sparkscale[1] = exp[1].evalF(c)
		}
	default:
		if !palFX(sc).runSub(c, &hd.palfx, paramID, exp) {
			return false
		}
	}
	return true
}

func (sc hitDef) Run(c *Char, _ []int32) bool {
	crun := c
	crun.hitdef.clear(crun, crun.localscl)
	crun.hitdef.playerNo = sys.workingState.playerNo
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if paramID == hitDef_redirectid {
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				crun.hitdef.clear(crun, crun.localscl)
				crun.hitdef.playerNo = sys.workingState.playerNo
			} else {
				return false
			}
		}
		// Mugen 1.1 behavior if invertblend param is omitted (Only if char mugenversion = 1.1)
		if c.stWgi().mugenver[0] == 1 && c.stWgi().mugenver[1] == 1 && c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
			crun.hitdef.palfx.invertblend = -2
		}
		sc.runSub(c, &crun.hitdef, paramID, exp)
		return true
	})
	// In WinMugen, when the attr of Hitdef is set to 'Throw' and the pausetime
	// on the attacker's side is greater than 1, it no longer executes every frame
	if c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 && c.stWgi().mugenver[0] != 1 && // Not crun
		crun.hitdef.attr&int32(AT_AT) != 0 && crun.hitdef.pausetime > 0 && crun.moveContact() == 1 { // crun
		crun.hitdef.attr = 0
		return false
	}
	crun.setHitdefDefault(&crun.hitdef)
	return false
}

type reversalDef hitDef

const (
	reversalDef_reversal_attr = iota + hitDef_last + 1
	reversalDef_redirectid
)

func (sc reversalDef) Run(c *Char, _ []int32) bool {
	crun := c
	crun.hitdef.clear(crun, crun.localscl)
	crun.hitdef.playerNo = sys.workingState.playerNo
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case reversalDef_reversal_attr:
			crun.hitdef.reversal_attr = exp[0].evalI(c)
		case reversalDef_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				crun.hitdef.clear(crun, crun.localscl)
				crun.hitdef.playerNo = sys.workingState.playerNo
			} else {
				return false
			}
		default:
			hitDef(sc).runSub(c, &crun.hitdef, paramID, exp)
		}
		return true
	})
	crun.setHitdefDefault(&crun.hitdef)
	return false
}

type projectile hitDef

const (
	projectile_postype = iota + hitDef_last + 1
	projectile_projid
	projectile_projremove
	projectile_projremovetime
	projectile_projshadow
	projectile_projmisstime
	projectile_projhits
	projectile_projpriority
	projectile_projhitanim
	projectile_projremanim
	projectile_projcancelanim
	projectile_velocity
	projectile_velmul
	projectile_remvelocity
	projectile_accel
	projectile_projscale
	projectile_projangle
	projectile_projclsnscale
	projectile_projclsnangle
	projectile_offset
	projectile_projsprpriority
	projectile_projlayerno
	projectile_projstagebound
	projectile_projedgebound
	projectile_projheightbound
	projectile_projdepthbound
	projectile_projanim
	projectile_supermovetime
	projectile_pausemovetime
	projectile_ownpal
	projectile_remappal
	projectile_projwindow
	projectile_projxshear
	// projectile_platform
	// projectile_platformwidth
	// projectile_platformheight
	// projectile_platformfence
	// projectile_platformangle
	projectile_redirectid
)

// Additions to this state controller should also be done to ModifyProjectile
func (sc projectile) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	var p *Projectile
	pt := PT_P1
	var x, y, z float32 = 0, 0, 0
	op := false
	clsnscale := false
	rp := [...]int32{-1, 0}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		if p == nil {
			if paramID == projectile_redirectid {
				if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
					crun = rid
					redirscale = c.localscl / crun.localscl
				} else {
					return false
				}
			}
			p = crun.newProj()
			if p == nil {
				return false
			}
		}
		switch paramID {
		case projectile_postype:
			pt = PosType(exp[0].evalI(c))
		case projectile_projid:
			p.id = exp[0].evalI(c)
		case projectile_projremove:
			p.remove = exp[0].evalB(c)
		case projectile_projremovetime:
			p.removetime = exp[0].evalI(c)
		case projectile_projshadow:
			p.shadow[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				p.shadow[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					p.shadow[2] = exp[2].evalI(c)
				}
			}
		case projectile_projmisstime:
			p.misstime = exp[0].evalI(c)
		case projectile_projhits:
			p.hits = exp[0].evalI(c)
		case projectile_projpriority:
			p.priority = exp[0].evalI(c)
			p.priorityPoints = p.priority
		case projectile_projhitanim:
			p.hitanim = exp[1].evalI(c)
			p.hitanim_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case projectile_projremanim:
			p.remanim = Max(-1, exp[1].evalI(c))
			p.remanim_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case projectile_projcancelanim:
			p.cancelanim = Max(-1, exp[1].evalI(c))
			p.cancelanim_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case projectile_velocity:
			p.velocity[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				p.velocity[1] = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					p.velocity[2] = exp[2].evalF(c) * redirscale
				}
			}
		case projectile_velmul:
			p.velmul[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				p.velmul[1] = exp[1].evalF(c)
				if len(exp) > 2 {
					p.velmul[2] = exp[2].evalF(c)
				}
			}
		case projectile_remvelocity:
			p.remvelocity[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				p.remvelocity[1] = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					p.remvelocity[2] = exp[2].evalF(c) * redirscale
				}
			}
		case projectile_accel:
			p.accel[0] = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				p.accel[1] = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					p.accel[2] = exp[2].evalF(c)
				}
			}
		case projectile_projscale:
			p.scale[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				p.scale[1] = exp[1].evalF(c)
			}
		case projectile_projangle:
			p.angle = exp[0].evalF(c)
		case projectile_offset:
			x = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				y = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					z = exp[2].evalF(c) * redirscale
				}
			}
		case projectile_projsprpriority:
			p.sprpriority = exp[0].evalI(c)
		case projectile_projlayerno:
			l := exp[0].evalI(c)
			if l > 0 {
				p.layerno = 1
			} else if l < 0 {
				p.layerno = -1
			} else {
				p.layerno = 0
			}
		case projectile_projstagebound:
			p.stagebound = int32(float32(exp[0].evalI(c)) * redirscale)
		case projectile_projedgebound:
			p.edgebound = int32(float32(exp[0].evalI(c)) * redirscale)
		case projectile_projheightbound:
			p.heightbound[0] = int32(float32(exp[0].evalI(c)) * redirscale)
			if len(exp) > 1 {
				p.heightbound[1] = int32(float32(exp[1].evalI(c)) * redirscale)
			}
		case projectile_projdepthbound:
			p.depthbound = int32(float32(exp[0].evalI(c)) * redirscale)
		case projectile_projanim:
			p.anim = exp[1].evalI(c)
			p.anim_ffx = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case projectile_supermovetime:
			p.supermovetime = exp[0].evalI(c)
			if p.supermovetime >= 0 {
				p.supermovetime = Max(p.supermovetime, p.supermovetime+1)
			}
		case projectile_pausemovetime:
			p.pausemovetime = exp[0].evalI(c)
			if p.pausemovetime >= 0 {
				p.pausemovetime = Max(p.pausemovetime, p.pausemovetime+1)
			}
		case projectile_ownpal:
			op = exp[0].evalB(c)
		case projectile_remappal:
			rp[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				rp[1] = exp[1].evalI(c)
			}
		case projectile_projclsnscale:
			clsnscale = true
			p.clsnScale[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				p.clsnScale[1] = exp[1].evalF(c)
			} else {
				p.clsnScale[1] = 1.0 // Default
			}
		case projectile_projclsnangle:
			p.clsnAngle = exp[0].evalF(c)
		case projectile_projwindow:
			p.window = [4]float32{exp[0].evalF(c) * redirscale, exp[1].evalF(c) * redirscale, exp[2].evalF(c) * redirscale, exp[3].evalF(c) * redirscale}
		case projectile_projxshear:
			p.xshear = exp[0].evalF(c)
		// case projectile_platform:
		// 	p.platform = exp[0].evalB(c)
		// case projectile_platformwidth:
		// 	p.platformWidth[0] = exp[0].evalF(c) * redirscale
		// 	if len(exp) > 1 {
		// 		p.platformWidth[1] = exp[1].evalF(c) * redirscale
		// 	}
		// case projectile_platformheight:
		// 	p.platformHeight[0] = exp[0].evalF(c) * redirscale
		// 	if len(exp) > 1 {
		// 		p.platformHeight[1] = exp[1].evalF(c) * redirscale
		// 	}
		// case projectile_platformangle:
		// 	p.platformAngle = exp[0].evalF(c)
		// case projectile_platformfence:
		// 	p.platformFence = exp[0].evalB(c)
		default:
			if !hitDef(sc).runSub(c, &p.hitdef, paramID, exp) {
				afterImage(sc).runSub(c, &p.aimg, paramID, exp)
			}
		}
		return true
	})
	if p == nil {
		return false
	}
	crun.setHitdefDefault(&p.hitdef)
	if p.hitanim == -1 {
		p.hitanim_ffx = p.anim_ffx
	}
	if p.remanim == IErr {
		p.remanim = p.hitanim
		p.remanim_ffx = p.hitanim_ffx
	}
	if p.cancelanim == IErr {
		p.cancelanim = p.remanim
		p.cancelanim_ffx = p.remanim_ffx
	}
	if p.aimg.time != 0 {
		p.aimg.setupPalFX()
	}
	crun.projInit(p, pt, x, y, z, op, rp[0], rp[1], clsnscale)
	return false
}

type modifyHitDef hitDef

const (
	modifyHitDef_redirectid = iota + hitDef_last + 1
)

func (sc modifyHitDef) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyHitDef_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		default:
			if crun.hitdef.attr > 0 && crun.hitdef.reversal_attr == 0 {
				hitDef(sc).runSub(c, &crun.hitdef, paramID, exp)
			}
		}
		return true
	})
	return false
}

type modifyReversalDef hitDef

const (
	modifyReversalDef_reversal_attr = iota + hitDef_last + 1
	modifyReversalDef_redirectid
)

func (sc modifyReversalDef) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyReversalDef_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		case modifyReversalDef_reversal_attr:
			if crun.hitdef.reversal_attr > 0 {
				crun.hitdef.reversal_attr = exp[0].evalI(c)
			}
		default:
			if crun.hitdef.reversal_attr > 0 {
				hitDef(sc).runSub(c, &crun.hitdef, paramID, exp)
			}
		}
		return true
	})
	return false
}

type modifyProjectile hitDef

const (
	modifyProjectile_redirectid = iota + hitDef_last + 1
	modifyProjectile_id
	modifyProjectile_index
)

func (sc modifyProjectile) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	mpid := int32(-1)
	mpidx := int32(-1)
	var projs []*Projectile
	eachProj := func(f func(p *Projectile)) {
		if mpidx < 0 {
			for _, p := range projs {
				f(p)
			}
		} else if mpidx < int32(len(projs)) {
			f(projs[mpidx])
		}
	}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyProjectile_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		case modifyProjectile_id: // ID's to modify
			mpid = exp[0].evalI(c)
		case modifyProjectile_index: // index to modify
			mpidx = exp[0].evalI(c)
		default:
			if crun.helperIndex != 0 {
				return false
			}
			if len(projs) == 0 {
				projs = crun.getProjs(mpid)
				if len(projs) == 0 {
					return false
				}
			}
			switch paramID {
			case projectile_projremove:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.remove = v1
				})
			case projectile_projremovetime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.removetime = v1
				})
			//case projectile_projshadow:
			case projectile_projmisstime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.misstime = v1
				})
			case projectile_projhits:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hits = v1
				})
			case projectile_projpriority:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.priority = v1
					p.priorityPoints = p.priority
				})
			case projectile_projhitanim:
				var v1 string
				var v2 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
				}
				eachProj(func(p *Projectile) {
					p.hitanim_ffx = v1
					p.hitanim = v2
				})
			case projectile_projremanim:
				var v1 string
				var v2 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = Max(-1, exp[1].evalI(c))
				}
				eachProj(func(p *Projectile) {
					p.remanim_ffx = v1
					p.remanim = v2
				})
			case projectile_projcancelanim:
				var v1 string
				var v2 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = Max(-1, exp[1].evalI(c))
				}
				eachProj(func(p *Projectile) {
					p.cancelanim_ffx = v1
					p.cancelanim = v2
				})
			case projectile_velocity:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.velocity[0] = v1 * redirscale
					p.velocity[1] = v2 * redirscale
					p.velocity[2] = v3 * redirscale
				})
			case projectile_velmul:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.velmul[0] = v1
					p.velmul[1] = v2
					p.velmul[2] = v3
				})
			case projectile_remvelocity:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.remvelocity[0] = v1 * redirscale
					p.remvelocity[1] = v2 * redirscale
					p.remvelocity[2] = v3 * redirscale
				})
			case projectile_accel:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.accel[0] = v1 * redirscale
					p.accel[1] = v2 * redirscale
					p.accel[2] = v3 * redirscale
				})
			case projectile_projscale:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					p.scale[0] = v1
					p.scale[1] = v2
				})
			case projectile_projangle:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.angle = v1
				})
			//case projectile_offset: // Pointless because it's only used when the projectile is created
			case projectile_projsprpriority:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.sprpriority = v1
				})
			case projectile_projlayerno:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					if v1 > 0 {
						p.layerno = 1
					} else if v1 < 0 {
						p.layerno = -1
					} else {
						p.layerno = 0
					}
				})
			case projectile_projstagebound:
				v1 := int32(float32(exp[0].evalI(c)) * redirscale)
				eachProj(func(p *Projectile) {
					p.stagebound = v1
				})
			case projectile_projedgebound:
				v1 := int32(float32(exp[0].evalI(c)) * redirscale)
				eachProj(func(p *Projectile) {
					p.edgebound = v1
				})
			case projectile_projheightbound:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					p.heightbound[0] = int32(v1 * redirscale)
					p.heightbound[1] = int32(v2 * redirscale)
				})
			case projectile_projdepthbound:
				v1 := int32(float32(exp[0].evalI(c)) * redirscale)
				eachProj(func(p *Projectile) {
					p.depthbound = v1
				})
			case projectile_projanim:
				var v1 string
				var v2 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = Max(-1, exp[1].evalI(c))
				}
				eachProj(func(p *Projectile) {
					if p.anim != v2 || p.anim_ffx != v1 {
						p.anim_ffx = v1
						p.anim = v2
						p.ani = c.getAnim(p.anim, p.anim_ffx, true) // need to change anim ref too
					}
				})
			case projectile_supermovetime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.supermovetime = v1
					if p.supermovetime >= 0 {
						p.supermovetime = Max(p.supermovetime, p.supermovetime+1)
					}
				})
			case projectile_pausemovetime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.pausemovetime = v1
					if p.pausemovetime >= 0 {
						p.pausemovetime = Max(p.pausemovetime, p.pausemovetime+1)
					}
				})
			//case projectile_ownpal: // TODO: Test these later. May cause issues
			//case projectile_remappal:
			case projectile_projclsnscale:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					p.clsnScale[0] = v1
					p.clsnScale[1] = v2
				})
			case projectile_projclsnangle:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.clsnAngle = v1
				})
			case projectile_projwindow:
				v1 := exp[0].evalF(c) * redirscale
				v2 := exp[1].evalF(c) * redirscale
				v3 := exp[2].evalF(c) * redirscale
				v4 := exp[3].evalF(c) * redirscale
				eachProj(func(p *Projectile) {
					p.window = [4]float32{v1, v2, v3, v4}
				})
			case projectile_projxshear:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.xshear = v1
				})
			case hitDef_attr:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.attr = v1
				})
			case hitDef_guardflag:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.guardflag = v1
				})
			case hitDef_hitflag:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.hitflag = v1
				})
			case hitDef_ground_type:
				v1 := HitType(exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.ground_type = v1
				})
			case hitDef_air_type:
				v1 := HitType(exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.air_type = v1
				})
			case hitDef_animtype:
				v1 := Reaction(exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.animtype = v1
				})
			case hitDef_air_animtype:
				v1 := Reaction(exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.air_animtype = v1
				})
			case hitDef_fall_animtype:
				v1 := Reaction(exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.fall_animtype = v1
				})
			case hitDef_affectteam:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.affectteam = v1
				})
			case hitDef_teamside:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					if v1 > 2 {
						p.hitdef.teamside = 2
					} else if v1 < 0 {
						p.hitdef.teamside = 0
					} else {
						p.hitdef.teamside = int(v1)
					}
				})
			case hitDef_id:
				v1 := Max(0, exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.id = v1
				})
			case hitDef_chainid:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.chainid = v1
				})
			case hitDef_nochainid:
				val := make([]int32, int(math.Min(8, float64(len(exp)))))
				for i := 0; i < len(val); i++ {
					val[i] = exp[i].evalI(c)
				}
				eachProj(func(p *Projectile) {
					for i := 0; i < len(val); i++ {
						p.hitdef.nochainid[i] = val[i]
					}
				})
			case hitDef_kill:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.kill = v1
				})
			case hitDef_guard_kill:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.guard_kill = v1
				})
			case hitDef_fall_kill:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_kill = v1
				})
			case hitDef_hitonce:
				v1 := Btoi(exp[0].evalB(c))
				eachProj(func(p *Projectile) {
					p.hitdef.hitonce = v1
				})
			case hitDef_air_juggle:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.air_juggle = v1
				})
			case hitDef_getpower:
				var v1, v2 int32
				v1 = Max(IErr+1, exp[0].evalI(c))
				if len(exp) > 1 {
					v2 = Max(IErr+1, exp[1].evalI(c))
				}
				eachProj(func(p *Projectile) {
					p.hitdef.hitgetpower = v1
					p.hitdef.guardgetpower = v2
				})
			case hitDef_damage:
				var v1, v2 int32
				v1 = exp[0].evalI(c)
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.hitdamage = v1
					p.hitdef.guarddamage = v2
				})
			case hitDef_givepower:
				var v1, v2 int32
				v1 = Max(IErr+1, exp[0].evalI(c))
				if len(exp) > 1 {
					v2 = Max(IErr+1, exp[1].evalI(c))
				}
				eachProj(func(p *Projectile) {
					p.hitdef.hitgivepower = v1
					p.hitdef.guardgivepower = v2
				})
			case hitDef_numhits:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.numhits = v1
				})
			case hitDef_hitsound:
				var v1 string
				var v2, v3 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
					if len(exp) > 2 {
						v3 = exp[2].evalI(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.hitsound_ffx = v1
					p.hitdef.hitsound[0] = v2
					p.hitdef.hitsound[1] = v3
				})
			case hitDef_hitsound_channel:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.hitsound_channel = v1
				})
			case hitDef_guardsound:
				var v1 string
				var v2, v3 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
					if len(exp) > 2 {
						v3 = exp[2].evalI(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.guardsound_ffx = v1
					p.hitdef.guardsound[0] = v2
					p.hitdef.guardsound[1] = v3
				})
			case hitDef_guardsound_channel:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.guardsound_channel = v1
				})
			case hitDef_priority:
				var v1 int32
				var v2 TradeType
				v1 = exp[0].evalI(c)
				if len(exp) > 1 {
					v2 = TradeType(exp[1].evalI(c))
				}
				eachProj(func(p *Projectile) {
					p.hitdef.priority = v1
					p.hitdef.prioritytype = v2
				})
			case hitDef_p1stateno:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.p1stateno = v1
				})
			case hitDef_p2stateno:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.p2stateno = v1
					p.hitdef.p2getp1state = true
				})
			case hitDef_p2getp1state:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.p2getp1state = v1
				})
			//case hitDef_p1sprpriority:
			//	p.hitdef.p1sprpriority = exp[0].evalI(c)
			case hitDef_p2sprpriority:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.p2sprpriority = v1
				})
			case hitDef_forcestand:
				v1 := Btoi(exp[0].evalB(c))
				eachProj(func(p *Projectile) {
					p.hitdef.forcestand = v1
				})
			case hitDef_forcecrouch:
				v1 := Btoi(exp[0].evalB(c))
				eachProj(func(p *Projectile) {
					p.hitdef.forcecrouch = v1
				})
			case hitDef_forcenofall:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.forcenofall = v1
				})
			case hitDef_fall_damage:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_damage = v1
				})
			case hitDef_fall_xvelocity:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_xvelocity = v1
				})
			case hitDef_fall_yvelocity:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_yvelocity = v1
				})
			case hitDef_fall_zvelocity:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_zvelocity = v1
				})
			case hitDef_fall_recover:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_recover = v1
				})
			case hitDef_fall_recovertime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_recovertime = v1
				})
			case hitDef_sparkno:
				var v1 string
				var v2 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.sparkno_ffx = v1
					p.hitdef.sparkno = v2
				})
			case hitDef_sparkangle:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.sparkangle = v1
				})
			case hitDef_guard_sparkno:
				var v1 string
				var v2 int32
				v1 = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.guard_sparkno_ffx = v1
					p.hitdef.guard_sparkno = v2
				})
			case hitDef_guard_sparkangle:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.guard_sparkangle = v1
				})
			case hitDef_sparkxy:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.sparkxy[0] = v1
					p.hitdef.sparkxy[1] = v2
				})
			case hitDef_down_hittime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.down_hittime = v1
				})
			//case hitDef_p1facing: // Doesn't work for projectiles
			//	p.hitdef.p1facing = exp[0].evalI(c)
			//case hitDef_p1getp2facing: // Doesn't work for projectiles
			case hitDef_mindist:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.mindist[0] = v1
					p.hitdef.mindist[1] = v2
					p.hitdef.mindist[2] = v3
				})
			case hitDef_maxdist:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.maxdist[0] = v1
					p.hitdef.maxdist[1] = v2
					p.hitdef.maxdist[2] = v3
				})
			case hitDef_snap:
				var v1, v2, v3 float32
				var v4 int32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
						if len(exp) > 3 {
							v4 = exp[2].evalI(c)
						}
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.snap[0] = v1
					p.hitdef.snap[1] = v2
					p.hitdef.snap[2] = v3
					p.hitdef.snaptime = v4
				})
			case hitDef_p2facing:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.p2facing = v1
				})
			case hitDef_air_hittime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.air_hittime = v1
				})
			case hitDef_fall:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.ground_fall = v1
				})
			case hitDef_air_fall:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.air_fall = v1
				})
			//case hitDef_air_cornerpush_veloff:
			//	p.hitdef.air_cornerpush_veloff = exp[0].evalF(c)
			case hitDef_down_bounce:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.down_bounce = v1
				})
			case hitDef_down_velocity:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.down_velocity[0] = v1
					p.hitdef.down_velocity[1] = v2
					p.hitdef.down_velocity[2] = v3
				})
			//case hitDef_down_cornerpush_veloff:
			//	p.hitdef.down_cornerpush_veloff = exp[0].evalF(c)
			case hitDef_ground_hittime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.ground_hittime = v1
				})
			case hitDef_guard_hittime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.guard_hittime = v1
				})
			case hitDef_guard_dist_x:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					if v1 >= 0 {
						p.hitdef.guard_dist_x[0] = v1
					}
					if v2 >= 0 {
						p.hitdef.guard_dist_x[1] = v2
					}
				})
			case hitDef_guard_dist_y:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					if v1 >= 0 {
						p.hitdef.guard_dist_y[0] = v1
					}
					if v2 >= 0 {
						p.hitdef.guard_dist_y[1] = v2
					}
				})
			case hitDef_guard_dist_z:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					if v1 >= 0 {
						p.hitdef.guard_dist_z[0] = v1
					}
					if v2 >= 0 {
						p.hitdef.guard_dist_z[1] = v2
					}
				})
			case hitDef_pausetime:
				var v1, v2 int32
				v1 = exp[0].evalI(c)
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.pausetime = v1
					p.hitdef.shaketime = v2
				})
			case hitDef_guard_pausetime:
				var v1, v2 int32
				v1 = exp[0].evalI(c)
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.guard_pausetime = v1
					p.hitdef.guard_shaketime = v2
				})
			case hitDef_air_velocity:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.air_velocity[0] = v1
					p.hitdef.air_velocity[1] = v2
					p.hitdef.air_velocity[2] = v3
				})
			case hitDef_airguard_velocity:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.airguard_velocity[0] = v1
					p.hitdef.airguard_velocity[1] = v2
					p.hitdef.airguard_velocity[2] = v3
				})
			case hitDef_ground_slidetime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.ground_slidetime = v1
				})
			case hitDef_guard_slidetime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.guard_slidetime = v1
				})
			case hitDef_guard_ctrltime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.guard_ctrltime = v1
				})
			case hitDef_airguard_ctrltime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.airguard_ctrltime = v1
				})
			case hitDef_ground_velocity_x:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.ground_velocity[0] = v1
				})
			case hitDef_ground_velocity_y:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.ground_velocity[1] = v1
				})
			case hitDef_ground_velocity_z:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.ground_velocity[2] = v1
				})
			case hitDef_guard_velocity:
				var v1, v2, v3 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
					if len(exp) > 2 {
						v3 = exp[2].evalF(c)
					}
				}
				eachProj(func(p *Projectile) {
					p.hitdef.guard_velocity[0] = v1
					p.hitdef.guard_velocity[1] = v2
					p.hitdef.guard_velocity[2] = v3
				})
			//case hitDef_ground_cornerpush_veloff:
			//	p.hitdef.ground_cornerpush_veloff = exp[0].evalF(c)
			//case hitDef_guard_cornerpush_veloff:
			//	p.hitdef.guard_cornerpush_veloff = exp[0].evalF(c)
			//case hitDef_airguard_cornerpush_veloff:
			//	p.hitdef.airguard_cornerpush_veloff = exp[0].evalF(c)
			case hitDef_xaccel:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.xaccel = v1
				})
			case hitDef_yaccel:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.yaccel = v1
				})
			case hitDef_zaccel:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.zaccel = v1
				})
			case hitDef_envshake_time:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.envshake_time = v1
				})
			case hitDef_envshake_ampl:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.envshake_ampl = v1
				})
			case hitDef_envshake_freq:
				v1 := MaxF(0, exp[0].evalF(c))
				eachProj(func(p *Projectile) {
					p.hitdef.envshake_freq = v1
				})
			case hitDef_envshake_phase:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.envshake_phase = v1
				})
			case hitDef_envshake_mul:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.envshake_mul = v1
				})
			case hitDef_envshake_dir:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.envshake_dir = v1
				})
			case hitDef_fall_envshake_time:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_envshake_time = v1
				})
			case hitDef_fall_envshake_ampl:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_envshake_ampl = v1
				})
			case hitDef_fall_envshake_freq:
				v1 := MaxF(0, exp[0].evalF(c))
				eachProj(func(p *Projectile) {
					p.hitdef.fall_envshake_freq = v1
				})
			case hitDef_fall_envshake_phase:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_envshake_phase = v1
				})
			case hitDef_fall_envshake_mul:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_envshake_mul = v1
				})
			case hitDef_fall_envshake_dir:
				v1 := exp[0].evalF(c)
				eachProj(func(p *Projectile) {
					p.hitdef.fall_envshake_dir = v1
				})
			case hitDef_dizzypoints:
				v1 := Max(IErr+1, exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.dizzypoints = v1
				})
			case hitDef_guardpoints:
				v1 := Max(IErr+1, exp[0].evalI(c))
				eachProj(func(p *Projectile) {
					p.hitdef.guardpoints = v1
				})
			case hitDef_redlife:
				var v1, v2 int32
				v1 = Max(IErr+1, exp[0].evalI(c))
				if len(exp) > 1 {
					v2 = exp[1].evalI(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.hitredlife = v1
					p.hitdef.guardredlife = v2
				})
			case hitDef_score:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.score[0] = v1
					p.hitdef.score[1] = v2
				})
			case hitDef_p2clsncheck:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					if v1 == 0 || v1 == 1 || v1 == 2 || v1 == 3 {
						p.hitdef.p2clsncheck = v1
					} else {
						p.hitdef.p2clsncheck = -1
					}
				})
			case hitDef_p2clsnrequire:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					if v1 == 1 || v1 == 2 || v1 == 3 {
						p.hitdef.p2clsnrequire = v1
					} else {
						p.hitdef.p2clsnrequire = 0
					}
				})
			case hitDef_down_recover:
				v1 := exp[0].evalB(c)
				eachProj(func(p *Projectile) {
					p.hitdef.down_recover = v1
				})
			case hitDef_down_recovertime:
				v1 := exp[0].evalI(c)
				eachProj(func(p *Projectile) {
					p.hitdef.down_recovertime = v1
				})
			case hitDef_attack_depth:
				var v1, v2 float32
				v1 = exp[0].evalF(c)
				if len(exp) > 1 {
					v2 = exp[1].evalF(c)
				}
				eachProj(func(p *Projectile) {
					p.hitdef.attack_depth[0] = v1
					p.hitdef.attack_depth[1] = v2
				})
			default:
				eachProj(func(p *Projectile) {
					if !hitDef(sc).runSub(c, &p.hitdef, paramID, exp) {
						afterImage(sc).runSub(c, &p.aimg, paramID, exp)
					}
				})
			}
		}
		return true
	})
	return false
}

type width StateControllerBase

const (
	width_edge byte = iota
	width_player
	width_value
	width_redirectid
)

func (sc width) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case width_player:
			var v1, v2 float32
			v1 = exp[0].evalF(c)
			if len(exp) > 1 {
				v2 = exp[1].evalF(c)
			}
			crun.setWidth(v1*redirscale, v2*redirscale)
		case width_edge:
			var v1, v2 float32
			v1 = exp[0].evalF(c)
			if len(exp) > 1 {
				v2 = exp[1].evalF(c)
			}
			crun.setWidthEdge(v1*redirscale, v2*redirscale)
		case width_value:
			var v1, v2 float32
			v1 = exp[0].evalF(c)
			if len(exp) > 1 {
				v2 = exp[1].evalF(c)
			}
			crun.setWidth(v1*redirscale, v2*redirscale)
			crun.setWidthEdge(v1*redirscale, v2*redirscale)
		case width_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = (320 / c.localcoord) / (320 / crun.localcoord)
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type sprPriority StateControllerBase

const (
	sprPriority_value byte = iota
	sprPriority_layerno
	sprPriority_redirectid
)

func (sc sprPriority) Run(c *Char, _ []int32) bool {
	crun := c
	v := int32(0) // Mugen uses 0 even if no value is set at all
	l := int32(0) // Defaults to 0 so that chars are less likely to be left forgotten in a different layer
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case sprPriority_value:
			v = exp[0].evalI(c)
		case sprPriority_layerno:
			l = exp[0].evalI(c)
		case sprPriority_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.sprPriority = v
	crun.layerNo = l
	return false
}

type varSet StateControllerBase

const (
	varSet_ byte = iota
	varSet_redirectid
)

func (sc varSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case varSet_:
			exp[0].run(crun)
		case varSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type turn StateControllerBase

const (
	turn_ byte = iota
	turn_redirectid
)

func (sc turn) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case turn_:
			crun.setFacing(-crun.facing)
		case turn_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type targetFacing StateControllerBase

const (
	targetFacing_id byte = iota
	targetFacing_index
	targetFacing_value
	targetFacing_redirectid
)

func (sc targetFacing) Run(c *Char, _ []int32) bool {
	crun := c
	tid, tidx := int32(-1), int(-1)
	var value int32
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetFacing_id:
			tid = exp[0].evalI(c)
		case targetFacing_index:
			tidx = int(exp[0].evalI(c))
		case targetFacing_value:
			value = exp[0].evalI(c)
		case targetFacing_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 && value != 0 {
		crun.targetFacing(tar, value)
	}
	return false
}

type targetBind StateControllerBase

const (
	targetBind_id byte = iota
	targetBind_index
	targetBind_time
	targetBind_pos
	targetBind_redirectid
)

func (sc targetBind) Run(c *Char, _ []int32) bool {
	crun := c
	tid, tidx := int32(-1), int(-1)
	var redirscale float32 = 1.0
	time := int32(1)
	var x, y, z float32 = 0, 0, 0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetBind_id:
			tid = exp[0].evalI(c)
		case targetBind_index:
			tidx = int(exp[0].evalI(c))
		case targetBind_time:
			time = exp[0].evalI(c)
		case targetBind_pos:
			x = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				y = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					z = exp[2].evalF(c) * redirscale
				}
			}
		case targetBind_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetBind(tar, time, x, y, z)
	}
	return false
}

type bindToTarget StateControllerBase

const (
	bindToTarget_id byte = iota
	bindToTarget_index
	bindToTarget_time
	bindToTarget_pos
	bindToTarget_posz
	bindToTarget_redirectid
)

func (sc bindToTarget) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	tid, tidx := int32(-1), int(0)
	time, x, y, z, hmf := int32(1), float32(0), float32(math.NaN()), float32(math.NaN()), HMF_F
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case bindToTarget_id:
			tid = exp[0].evalI(c)
		case bindToTarget_index:
			tidx = int(exp[0].evalI(c))
		case bindToTarget_time:
			time = exp[0].evalI(c)
		case bindToTarget_pos:
			x = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				y = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					hmf = HMF(exp[2].evalI(c))
				}
			}
		case bindToTarget_posz:
			z = exp[0].evalF(c) * redirscale
		case bindToTarget_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.bindToTarget(tar, time, x, y, z, hmf)
	}
	return false
}

type targetLifeAdd StateControllerBase

const (
	targetLifeAdd_id byte = iota
	targetLifeAdd_index
	targetLifeAdd_absolute
	targetLifeAdd_kill
	targetLifeAdd_dizzy
	targetLifeAdd_redlife
	targetLifeAdd_value
	targetLifeAdd_redirectid
)

func (sc targetLifeAdd) Run(c *Char, _ []int32) bool {
	crun := c
	abs, kill, d, r := false, true, true, true
	var value int32
	tid, tidx := int32(-1), int(-1)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetLifeAdd_id:
			tid = exp[0].evalI(c)
		case targetLifeAdd_index:
			tidx = int(exp[0].evalI(c))
		case targetLifeAdd_absolute:
			abs = exp[0].evalB(c)
		case targetLifeAdd_kill:
			kill = exp[0].evalB(c)
		case targetLifeAdd_dizzy:
			d = exp[0].evalB(c)
		case targetLifeAdd_redlife:
			r = exp[0].evalB(c)
		case targetLifeAdd_value:
			value = exp[0].evalI(c)
		case targetLifeAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetLifeAdd(tar, value, kill, abs, d, r)
	}
	return false
}

type targetState StateControllerBase

const (
	targetState_id byte = iota
	targetState_index
	targetState_value
	targetState_redirectid
)

func (sc targetState) Run(c *Char, _ []int32) bool {
	crun := c
	tid, tidx := int32(-1), int(-1)
	vl := int32(-1)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetState_id:
			tid = exp[0].evalI(c)
		case targetState_index:
			tidx = int(exp[0].evalI(c))
		case targetState_value:
			vl = exp[0].evalI(c)
		case targetState_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetState(tar, vl)
	}
	return false
}

type targetVelSet StateControllerBase

const (
	targetVelSet_id byte = iota
	targetVelSet_index
	targetVelSet_x
	targetVelSet_y
	targetVelSet_z
	targetVelSet_redirectid
)

func (sc targetVelSet) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	tid, tidx := int32(-1), int(-1)
	var setx, sety, setz bool
	var velx, vely, velz float32
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetVelSet_id:
			tid = exp[0].evalI(c)
		case targetVelSet_index:
			tidx = int(exp[0].evalI(c))
		case targetVelSet_x:
			velx = exp[0].evalF(c) * redirscale
			setx = true
		case targetVelSet_y:
			vely = exp[0].evalF(c) * redirscale
			sety = true
		case targetVelSet_z:
			velz = exp[0].evalF(c) * redirscale
			setz = true
		case targetVelSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		if setx {
			crun.targetVelSetX(tar, velx)
		}
		if sety {
			crun.targetVelSetY(tar, vely)
		}
		if setz {
			crun.targetVelSetZ(tar, velz)
		}
	}
	return false
}

type targetVelAdd StateControllerBase

const (
	targetVelAdd_id byte = iota
	targetVelAdd_index
	targetVelAdd_x
	targetVelAdd_y
	targetVelAdd_z
	targetVelAdd_redirectid
)

func (sc targetVelAdd) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	tid, tidx := int32(-1), int(-1)
	var setx, sety, setz bool
	var velx, vely, velz float32
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetVelAdd_id:
			tid = exp[0].evalI(c)
		case targetVelAdd_index:
			tidx = int(exp[0].evalI(c))
		case targetVelAdd_x:
			velx = exp[0].evalF(c) * redirscale
			setx = true
		case targetVelAdd_y:
			vely = exp[0].evalF(c) * redirscale
			sety = true
		case targetVelAdd_z:
			velz = exp[0].evalF(c) * redirscale
			setz = true
		case targetVelAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		if setx {
			crun.targetVelAddX(tar, velx)
		}
		if sety {
			crun.targetVelAddY(tar, vely)
		}
		if setz {
			crun.targetVelAddZ(tar, velz)
		}
	}
	return false
}

type targetPowerAdd StateControllerBase

const (
	targetPowerAdd_id byte = iota
	targetPowerAdd_index
	targetPowerAdd_value
	targetPowerAdd_redirectid
)

func (sc targetPowerAdd) Run(c *Char, _ []int32) bool {
	crun := c
	tid, tidx := int32(-1), int(-1)
	vl := int32(0)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetPowerAdd_id:
			tid = exp[0].evalI(c)
		case targetPowerAdd_index:
			tidx = int(exp[0].evalI(c))
		case targetPowerAdd_value:
			vl = exp[0].evalI(c)
		case targetPowerAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetPowerAdd(tar, vl)
	}
	return false
}

type targetDrop StateControllerBase

const (
	targetDrop_excludeid byte = iota
	targetDrop_keepone
	targetDrop_redirectid
)

func (sc targetDrop) Run(c *Char, _ []int32) bool {
	crun := c
	eid, keep := int32(-1), true
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetDrop_excludeid:
			eid = exp[0].evalI(c)
		case targetDrop_keepone:
			keep = exp[0].evalB(c)
		case targetDrop_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(-1, -1)
	if len(tar) > 0 {
		crun.targetDrop(eid, -1, keep)
	}
	return false
}

type lifeAdd StateControllerBase

const (
	lifeAdd_absolute byte = iota
	lifeAdd_kill
	lifeAdd_value
	lifeAdd_redirectid
)

func (sc lifeAdd) Run(c *Char, _ []int32) bool {
	abs, kill := false, true
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case lifeAdd_absolute:
			abs = exp[0].evalB(c)
		case lifeAdd_kill:
			kill = exp[0].evalB(c)
		case lifeAdd_value:
			v := exp[0].evalI(c)
			// Mugen forces absolute parameter when healing characters
			if v > 0 && c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
				abs = true
			}
			crun.lifeAdd(float64(v), kill, abs)
			crun.ghv.kill = kill // The kill GetHitVar must currently be set here because c.lifeAdd is also used internally
		case lifeAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type lifeSet StateControllerBase

const (
	lifeSet_value byte = iota
	lifeSet_redirectid
)

func (sc lifeSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case lifeSet_value:
			crun.lifeSet(exp[0].evalI(c))
		case lifeSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type powerAdd StateControllerBase

const (
	powerAdd_value byte = iota
	powerAdd_redirectid
)

func (sc powerAdd) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case powerAdd_value:
			crun.powerAdd(exp[0].evalI(c))
		case powerAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type powerSet StateControllerBase

const (
	powerSet_value byte = iota
	powerSet_redirectid
)

func (sc powerSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case powerSet_value:
			crun.powerSet(exp[0].evalI(c))
		case powerSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type hitVelSet StateControllerBase

const (
	hitVelSet_x byte = iota
	hitVelSet_y
	hitVelSet_z
	hitVelSet_redirectid
)

func (sc hitVelSet) Run(c *Char, _ []int32) bool {
	// Note: HitVelSet doesn't require Movetype H in Mugen
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitVelSet_x:
			if exp[0].evalB(c) {
				crun.vel[0] = crun.ghv.xvel * crun.facing
			}
		case hitVelSet_y:
			if exp[0].evalB(c) {
				crun.vel[1] = crun.ghv.yvel
			}
		case hitVelSet_z:
			if exp[0].evalB(c) {
				crun.vel[2] = crun.ghv.zvel
			}
		case hitVelSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type screenBound StateControllerBase

const (
	screenBound_value byte = iota
	screenBound_movecamera
	screenBound_stagebound
	screenBound_redirectid
)

func (sc screenBound) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case screenBound_value:
			if exp[0].evalB(c) {
				crun.setCSF(CSF_screenbound)
			} else {
				crun.unsetCSF(CSF_screenbound)
			}
		case screenBound_movecamera:
			if exp[0].evalB(c) {
				crun.setCSF(CSF_movecamera_x)
			} else {
				crun.unsetCSF(CSF_movecamera_x)
			}
			if len(exp) > 1 {
				if exp[1].evalB(c) {
					crun.setCSF(CSF_movecamera_y)
				} else {
					crun.unsetCSF(CSF_movecamera_y)
				}
			} else {
				crun.unsetCSF(CSF_movecamera_y)
			}
		case screenBound_stagebound:
			if exp[0].evalB(c) {
				crun.setCSF(CSF_stagebound)
			} else {
				crun.unsetCSF(CSF_stagebound)
			}
		case screenBound_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type posFreeze StateControllerBase

const (
	posFreeze_value byte = iota
	posFreeze_redirectid
)

func (sc posFreeze) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case posFreeze_value:
			if exp[0].evalB(c) {
				crun.setCSF(CSF_posfreeze)
			}
		case posFreeze_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type envShake StateControllerBase

const (
	envShake_time byte = iota
	envShake_ampl
	envShake_freq
	envShake_mul
	envShake_phase
	envShake_dir
)

func (sc envShake) Run(c *Char, _ []int32) bool {
	sys.envShake.clear()
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case envShake_time:
			sys.envShake.time = exp[0].evalI(c)
		case envShake_ampl:
			sys.envShake.ampl = float32(int32(float32(exp[0].evalI(c)) * c.localscl))
			// Because of how localscl works, the amplitude will be slightly smaller during widescreen
			// This also happens in Mugen however
		case envShake_freq:
			sys.envShake.freq = MaxF(0, exp[0].evalF(c)*float32(math.Pi)/180)
		case envShake_phase:
			sys.envShake.phase = MaxF(0, exp[0].evalF(c)*float32(math.Pi)/180)
		case envShake_mul:
			sys.envShake.mul = exp[0].evalF(c)
		case envShake_dir:
			sys.envShake.dir = MaxF(0, exp[0].evalF(c)*float32(math.Pi)/180)
		}
		return true
	})
	sys.envShake.setDefaultPhase()
	return false
}

type hitOverride StateControllerBase

const (
	hitOverride_attr byte = iota
	hitOverride_slot
	hitOverride_stateno
	hitOverride_time
	hitOverride_forceair
	hitOverride_keepstate
	hitOverride_redirectid
)

func (sc hitOverride) Run(c *Char, _ []int32) bool {
	crun := c
	var a, s, st, t int32 = 0, 0, -1, 1
	f := false
	ks := false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitOverride_attr:
			a = exp[0].evalI(c)
		case hitOverride_slot:
			s = Max(0, exp[0].evalI(c))
			if s > 7 {
				s = 0
			}
		case hitOverride_stateno:
			st = exp[0].evalI(c)
		case hitOverride_time:
			t = exp[0].evalI(c)
			if t < -1 || t == 0 {
				t = 1
			}
		case hitOverride_forceair:
			f = exp[0].evalB(c)
		case hitOverride_keepstate:
			if st == -1 { // StateNo disables KeepState
				ks = exp[0].evalB(c)
			}
		case hitOverride_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if st < 0 && !ks {
		t = 0
	}
	pn := crun.playerNo
	crun.ho[s] = HitOverride{attr: a, stateno: st, time: t, forceair: f, keepState: ks, playerNo: pn}
	return false
}

type pause StateControllerBase

const (
	pause_time byte = iota
	pause_movetime
	pause_pausebg
	pause_endcmdbuftime
	pause_redirectid
)

func (sc pause) Run(c *Char, _ []int32) bool {
	crun := c
	var t, mt int32 = 0, 0
	sys.pausebg, sys.pauseendcmdbuftime = true, 0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case pause_time:
			t = exp[0].evalI(c)
		case pause_movetime:
			mt = exp[0].evalI(c)
		case pause_pausebg:
			sys.pausebg = exp[0].evalB(c)
		case pause_endcmdbuftime:
			sys.pauseendcmdbuftime = exp[0].evalI(c)
		case pause_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.setPauseTime(t, mt)
	return false
}

type superPause StateControllerBase

const (
	superPause_time byte = iota
	superPause_movetime
	superPause_pausebg
	superPause_endcmdbuftime
	superPause_darken
	superPause_anim
	superPause_pos
	superPause_p2defmul
	superPause_poweradd
	superPause_unhittable
	superPause_sound
	superPause_redirectid
)

func (sc superPause) Run(c *Char, _ []int32) bool {
	crun := c
	var t, mt int32 = 30, 0
	uh := true
	animset := false
	sys.superpmap.remap = nil
	sys.superpos = [2]float32{crun.pos[0] * crun.localscl, crun.pos[1] * crun.localscl}
	sys.superscale = [2]float32{crun.facing, 1}
	sys.superpausebg = true
	sys.superendcmdbuftime = 0
	sys.superdarken = true
	sys.superp2defmul = crun.gi().constants["super.targetdefencemul"]
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case superPause_time:
			t = exp[0].evalI(c)
		case superPause_movetime:
			mt = exp[0].evalI(c)
		case superPause_pausebg:
			sys.superpausebg = exp[0].evalB(c)
		case superPause_endcmdbuftime:
			sys.superendcmdbuftime = exp[0].evalI(c)
		case superPause_darken:
			sys.superdarken = exp[0].evalB(c)
		case superPause_anim:
			ffx := string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			num := exp[1].evalI(c)
			animset = true
			if sys.superanim = crun.getAnim(num, ffx, true); sys.superanim != nil {
				if ffx != "" && ffx != "s" {
					sys.superpmap.remap = nil
				} else {
					sys.superpmap.remap = crun.getPalMap()
				}
			}
		case superPause_pos:
			sys.superpos[0] += crun.facing * exp[0].evalF(c) * c.localscl
			if len(exp) > 1 {
				sys.superpos[1] += exp[1].evalF(c) * c.localscl
			}
		case superPause_p2defmul:
			sys.superp2defmul = exp[0].evalF(c)
			if sys.superp2defmul == 0 {
				sys.superp2defmul = crun.gi().constants["super.targetdefencemul"]
			}
		case superPause_poweradd:
			crun.powerAdd(exp[0].evalI(c))
		case superPause_unhittable:
			uh = exp[0].evalB(c)
		case superPause_sound:
			n := int32(0)
			if len(exp) > 2 {
				n = exp[2].evalI(c)
			}
			vo := int32(100)
			ffx := string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			crun.playSound(ffx, false, 0, exp[1].evalI(c), n, -1,
				vo, 0, 1, 1, nil, false, 0, 0, 0, 0, false, false)
		case superPause_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				sys.superpmap.remap = nil
				sys.superpos = [2]float32{crun.pos[0] * crun.localscl, crun.pos[1] * crun.localscl}
				sys.superscale = [2]float32{crun.facing, 1}
			} else {
				return false
			}
		}
		return true
	})
	if !animset {
		sys.superanim = crun.getAnim(100, "f", true) // Default animation
	}
	if sys.superanim != nil {
		sys.superanim.start_scale[0] *= crun.localscl
		sys.superanim.start_scale[1] *= crun.localscl
		// Apply Z axis perspective
		if sys.zEnabled() {
			sys.superpos = sys.drawposXYfromZ(sys.superpos, crun.localscl, crun.interPos[2], crun.zScale)
			sys.superscale[0] *= crun.zScale
			sys.superscale[1] *= crun.zScale
		}
	}
	crun.setSuperPauseTime(t, mt, uh)
	return false
}

type trans StateControllerBase

const (
	trans_trans byte = iota
	trans_redirectid
)

func (sc trans) Run(c *Char, _ []int32) bool {
	crun := c
	crun.alpha[1] = 255
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case trans_trans:
			crun.alpha[0] = exp[0].evalI(c)
			crun.alpha[1] = exp[1].evalI(c)
			if len(exp) >= 3 {
				crun.alpha[0] = Clamp(crun.alpha[0], 0, 255)
				crun.alpha[1] = Clamp(crun.alpha[1], 0, 255)
				//if len(exp) >= 4 {
				//	crun.alpha[1] = ^crun.alpha[1]
				//} else if crun.alpha[0] == 1 && crun.alpha[1] == 255 {
				if crun.alpha[0] == 1 && crun.alpha[1] == 255 {
					crun.alpha[0] = 0
				}
			}
		case trans_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.setCSF(CSF_trans)
	return false
}

type playerPush StateControllerBase

const (
	playerPush_value byte = iota
	playerPush_priority
	playerPush_redirectid
)

func (sc playerPush) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case playerPush_value:
			if exp[0].evalB(c) {
				crun.setCSF(CSF_playerpush)
			} else {
				crun.unsetCSF(CSF_playerpush)
			}
		case playerPush_priority:
			crun.pushPriority = exp[0].evalI(c)
		case playerPush_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type stateTypeSet StateControllerBase

const (
	stateTypeSet_statetype byte = iota
	stateTypeSet_movetype
	stateTypeSet_physics
	stateTypeSet_redirectid
)

func (sc stateTypeSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case stateTypeSet_statetype:
			crun.ss.changeStateType(StateType(exp[0].evalI(c)))
		case stateTypeSet_movetype:
			crun.ss.changeMoveType(MoveType(exp[0].evalI(c)))
		case stateTypeSet_physics:
			crun.ss.physics = StateType(exp[0].evalI(c))
		case stateTypeSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type angleDraw StateControllerBase

const (
	angleDraw_value byte = iota
	angleDraw_scale
	angleDraw_redirectid
)

func (sc angleDraw) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case angleDraw_value:
			crun.angleSet(exp[0].evalF(c))
		case angleDraw_scale:
			crun.angleDrawScale[0] *= exp[0].evalF(c)
			if len(exp) > 1 {
				crun.angleDrawScale[1] *= exp[1].evalF(c)
			}
		case angleDraw_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.setCSF(CSF_angledraw)
	return false
}

type angleSet StateControllerBase

const (
	angleSet_value byte = iota
	angleSet_redirectid
)

func (sc angleSet) Run(c *Char, _ []int32) bool {
	crun := c
	v := float32(0) // Mugen uses 0 if no value is set at all
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case angleSet_value:
			v = exp[0].evalF(c)
		case angleSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.angleSet(v)
	return false
}

type angleAdd StateControllerBase

const (
	angleAdd_value byte = iota
	angleAdd_redirectid
)

func (sc angleAdd) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case angleAdd_value:
			crun.angleSet(crun.angle + exp[0].evalF(c))
		case angleAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type angleMul StateControllerBase

const (
	angleMul_value byte = iota
	angleMul_redirectid
)

func (sc angleMul) Run(c *Char, _ []int32) bool {
	crun := c
	v := float32(0) // Mugen uses 0 if no value is set at all
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case angleMul_value:
			v = exp[0].evalF(c)
		case angleMul_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.angleSet(crun.angle * v)
	return false
}

type envColor StateControllerBase

const (
	envColor_value byte = iota
	envColor_time
	envColor_under
)

func (sc envColor) Run(c *Char, _ []int32) bool {
	sys.envcol = [...]int32{255, 255, 255}
	sys.envcol_time = 1
	sys.envcol_under = false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case envColor_value:
			sys.envcol[0] = exp[0].evalI(c)
			sys.envcol[1] = exp[1].evalI(c)
			sys.envcol[2] = exp[2].evalI(c)
		case envColor_time:
			sys.envcol_time = exp[0].evalI(c)
		case envColor_under:
			sys.envcol_under = exp[0].evalB(c)
		}
		return true
	})
	return false
}

type displayToClipboard StateControllerBase

const (
	displayToClipboard_params byte = iota
	displayToClipboard_text
	displayToClipboard_redirectid
)

func (sc displayToClipboard) Run(c *Char, _ []int32) bool {
	crun := c
	params := []interface{}{}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case displayToClipboard_params:
			for _, e := range exp {
				if bv := e.run(c); bv.vtype == VT_Float {
					params = append(params, bv.ToF())
				} else {
					params = append(params, bv.ToI())
				}
			}
		case displayToClipboard_text:
			crun.clipboardText = nil
			crun.appendToClipboard(sys.workingState.playerNo,
				int(exp[0].evalI(c)), params...)
		case displayToClipboard_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type appendToClipboard displayToClipboard

func (sc appendToClipboard) Run(c *Char, _ []int32) bool {
	crun := c
	params := []interface{}{}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case displayToClipboard_params:
			for _, e := range exp {
				if bv := e.run(c); bv.vtype == VT_Float {
					params = append(params, bv.ToF())
				} else {
					params = append(params, bv.ToI())
				}
			}
		case displayToClipboard_text:
			crun.appendToClipboard(sys.workingState.playerNo,
				int(exp[0].evalI(c)), params...)
		case displayToClipboard_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type clearClipboard StateControllerBase

const (
	clearClipboard_ byte = iota
	clearClipboard_redirectid
)

func (sc clearClipboard) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case clearClipboard_:
			crun.clipboardText = nil
		case clearClipboard_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type makeDust StateControllerBase

const (
	makeDust_spacing byte = iota
	makeDust_pos
	makeDust_pos2
	makeDust_redirectid
)

func (sc makeDust) Run(c *Char, _ []int32) bool {
	crun := c
	spacing := int(3) // Default spacing is 3
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case makeDust_spacing:
			spacing = int(exp[0].evalI(c))
		case makeDust_pos:
			x, y, z := exp[0].evalF(c), float32(0), float32(0)
			if len(exp) > 1 {
				y = exp[1].evalF(c)
				if len(exp) > 2 {
					z = exp[2].evalF(c)
				}
			}
			crun.makeDust(x-float32(crun.size.draw.offset[0]),
				y-float32(crun.size.draw.offset[1]), z, spacing)
		case makeDust_pos2:
			x, y, z := exp[0].evalF(c), float32(0), float32(0)
			if len(exp) > 1 {
				y = exp[1].evalF(c)
				if len(exp) > 2 {
					z = exp[2].evalF(c)
				}
			}
			crun.makeDust(x-float32(crun.size.draw.offset[0]),
				y-float32(crun.size.draw.offset[1]), z, spacing)
		case makeDust_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type attackDist StateControllerBase

const (
	attackDist_x byte = iota
	attackDist_y
	attackDist_z
	attackDist_redirectid
)

func (sc attackDist) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case attackDist_x:
			crun.hitdef.guard_dist_x[0] = MaxF(0, exp[0].evalF(c)*redirscale)
			if len(exp) > 1 {
				crun.hitdef.guard_dist_x[1] = MaxF(0, exp[1].evalF(c)*redirscale)
			}
			// It used to be that Ikemen AttackDist used a separate variable
			// However it was found that Mugen AttackDist modifies the HitDef directly just like this
			// https://github.com/ikemen-engine/Ikemen-GO/issues/2358
		case attackDist_y:
			crun.hitdef.guard_dist_y[0] = MaxF(0, exp[0].evalF(c)*redirscale)
			if len(exp) > 1 {
				crun.hitdef.guard_dist_y[1] = MaxF(0, exp[1].evalF(c)*redirscale)
			}
		case attackDist_z:
			crun.hitdef.guard_dist_z[0] = MaxF(0, exp[0].evalF(c)*redirscale)
			if len(exp) > 1 {
				crun.hitdef.guard_dist_z[1] = MaxF(0, exp[1].evalF(c)*redirscale)
			}
		case attackDist_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type attackMulSet StateControllerBase

const (
	attackMulSet_value byte = iota
	attackMulSet_damage
	attackMulSet_redlife
	attackMulSet_dizzypoints
	attackMulSet_guardpoints
	attackMulSet_redirectid
)

func (sc attackMulSet) Run(c *Char, _ []int32) bool {
	crun := c
	base := float32(crun.gi().data.attack) * crun.ocd().attackRatio / 100
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case attackMulSet_value:
			v := exp[0].evalF(c)
			crun.attackMul[0] = v * base
			crun.attackMul[1] = v * base
			crun.attackMul[2] = v * base
			crun.attackMul[3] = v * base
		case attackMulSet_damage:
			crun.attackMul[0] = exp[0].evalF(c) * base
		case attackMulSet_redlife:
			crun.attackMul[1] = exp[0].evalF(c) * base
		case attackMulSet_dizzypoints:
			crun.attackMul[2] = exp[0].evalF(c) * base
		case attackMulSet_guardpoints:
			crun.attackMul[3] = exp[0].evalF(c) * base
		case attackMulSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type defenceMulSet StateControllerBase

const (
	defenceMulSet_value byte = iota
	defenceMulSet_onHit
	defenceMulSet_mulType
	defenceMulSet_redirectid
)

func (sc defenceMulSet) Run(c *Char, _ []int32) bool {
	crun := c
	var val float32 = 1
	var onHit bool = false
	var mulType int32 = 1

	// Change default behavior for Mugen chars
	if c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
		onHit = true
		mulType = 0
	}

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case defenceMulSet_value:
			val = exp[0].evalF(c)
		case defenceMulSet_onHit:
			onHit = exp[0].evalB(c)
		case defenceMulSet_mulType:
			mulType = exp[0].evalI(c)
		case defenceMulSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})

	// Apply "value" according to "mulType"
	if mulType != 0 {
		crun.customDefense = val
	} else {
		crun.customDefense = 1.0 / val
	}

	// Apply "onHit"
	crun.defenseMulDelay = onHit

	return false
}

type fallEnvShake StateControllerBase

const (
	fallEnvShake_ byte = iota
	fallEnvShake_redirectid
)

func (sc fallEnvShake) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case fallEnvShake_:
			if crun.ghv.fall_envshake_time > 0 {
				sys.envShake = EnvShake{time: crun.ghv.fall_envshake_time,
					freq:  crun.ghv.fall_envshake_freq * math.Pi / 180,
					ampl:  float32(crun.ghv.fall_envshake_ampl) * c.localscl,
					phase: crun.ghv.fall_envshake_phase,
					mul:   crun.ghv.fall_envshake_mul,
					dir:   crun.ghv.fall_envshake_dir * float32(math.Pi) / 180}
				sys.envShake.setDefaultPhase()
				crun.ghv.fall_envshake_time = 0
			}
		case fallEnvShake_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type hitFallDamage StateControllerBase

const (
	hitFallDamage_ byte = iota
	hitFallDamage_redirectid
)

func (sc hitFallDamage) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitFallDamage_:
			crun.hitFallDamage()
		case hitFallDamage_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type hitFallVel StateControllerBase

const (
	hitFallVel_ byte = iota
	hitFallVel_redirectid
)

func (sc hitFallVel) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitFallVel_:
			crun.hitFallVel()
		case hitFallVel_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type hitFallSet StateControllerBase

const (
	hitFallSet_value byte = iota
	hitFallSet_xvel
	hitFallSet_yvel
	hitFallSet_zvel
	hitFallSet_redirectid
)

func (sc hitFallSet) Run(c *Char, _ []int32) bool {
	crun := c
	f, xv, yv, zv := int32(-1), float32(math.NaN()), float32(math.NaN()), float32(math.NaN())
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitFallSet_value:
			f = exp[0].evalI(c)
			if len(crun.ghv.hitBy) == 0 {
				return false
			}
		case hitFallSet_xvel:
			xv = exp[0].evalF(c)
		case hitFallSet_yvel:
			yv = exp[0].evalF(c)
		case hitFallSet_zvel:
			zv = exp[0].evalF(c)
		case hitFallSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.hitFallSet(f, xv, yv, zv)
	return false
}

type varRangeSet StateControllerBase

const (
	varRangeSet_first byte = iota
	varRangeSet_last
	varRangeSet_value
	varRangeSet_fvalue
	varRangeSet_redirectid
)

func (sc varRangeSet) Run(c *Char, _ []int32) bool {
	crun := c
	first := int32(0)
	last := int32(59) // Legacy default because MaxInt32 would stall the game

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case varRangeSet_first:
			first = exp[0].evalI(c)
		case varRangeSet_last:
			last = exp[0].evalI(c)
		case varRangeSet_value:
			val := exp[0].evalI(c)
			crun.varRangeSet(first, last, val)
		case varRangeSet_fvalue:
			fval := exp[0].evalF(c)
			crun.fvarRangeSet(first, last, fval)
		case varRangeSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type remapPal StateControllerBase

const (
	remapPal_source byte = iota
	remapPal_dest
	remapPal_redirectid
)

func (sc remapPal) Run(c *Char, _ []int32) bool {
	crun := c
	src := [...]int32{-1, 0}
	dst := [...]int32{-1, 0} // This is the default but technically the compiler crashes if dest is not specified
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case remapPal_source:
			src[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				src[1] = exp[1].evalI(c)
			}
		case remapPal_dest:
			dst = [...]int32{exp[0].evalI(c), -1}
			if len(exp) > 1 {
				dst[1] = exp[1].evalI(c)
			}
		case remapPal_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.remapPal(crun.getPalfx(), src, dst)
	return false
}

type stopSnd StateControllerBase

const (
	stopSnd_channel byte = iota
	stopSnd_redirectid
)

func (sc stopSnd) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case stopSnd_channel:
			if ch := Min(255, exp[0].evalI(c)); ch < 0 {
				sys.stopAllSound()
			} else if c := crun.soundChannels.Get(ch); c != nil {
				c.Stop()
			}
		case stopSnd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type sndPan StateControllerBase

const (
	sndPan_channel byte = iota
	sndPan_pan
	sndPan_abspan
	sndPan_redirectid
)

func (sc sndPan) Run(c *Char, _ []int32) bool {
	crun := c
	ch, pan, x := int32(-1), float32(0), &crun.pos[0]
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case sndPan_channel:
			ch = exp[0].evalI(c)
		case sndPan_pan:
			pan = exp[0].evalF(c)
		case sndPan_abspan:
			pan = exp[0].evalF(c)
			x = nil
		case sndPan_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				x = &crun.pos[0]
			} else {
				return false
			}
		}
		return true
	})
	if c := crun.soundChannels.Get(ch); c != nil {
		c.SetPan(pan*crun.facing, crun.localscl, x)
	}
	return false
}

type varRandom StateControllerBase

const (
	varRandom_v byte = iota
	varRandom_range
	varRandom_redirectid
)

func (sc varRandom) Run(c *Char, _ []int32) bool {
	crun := c
	var v int32
	var min, max int32 = 0, 1000
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case varRandom_v:
			v = exp[0].evalI(c)
		case varRandom_range:
			min, max = 0, exp[0].evalI(c)
			if len(exp) > 1 {
				min, max = max, exp[1].evalI(c)
			}
		case varRandom_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.varSet(v, RandI(min, max))
	return false
}

type gravity StateControllerBase

const (
	gravity_ byte = iota
	gravity_redirectid
)

func (sc gravity) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case gravity_:
			crun.gravity()
		case gravity_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type bindToParent StateControllerBase

const (
	bindToParent_time byte = iota
	bindToParent_facing
	bindToParent_pos
	bindToParent_redirectid
)

func (sc bindToParent) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	var x, y, z float32 = 0, 0, 0
	var time int32 = 1

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case bindToParent_time:
			time = exp[0].evalI(c)
		case bindToParent_facing:
			if f := exp[0].evalI(c); f < 0 {
				crun.bindFacing = -1
			} else if f > 0 {
				crun.bindFacing = 1
			}
		case bindToParent_pos:
			x = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				y = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					z = exp[2].evalF(c) * redirscale
				}
			}
		case bindToParent_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})

	p := crun.parent(true)
	if p == nil {
		return false
	}

	crun.bindPos[0] = x
	crun.bindPos[1] = y
	crun.bindPos[2] = z
	crun.setBindToId(p, false)
	crun.setBindTime(time)
	return false
}

type bindToRoot bindToParent

func (sc bindToRoot) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	var x, y, z float32 = 0, 0, 0
	var time int32 = 1

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case bindToParent_time:
			time = exp[0].evalI(c)
		case bindToParent_facing:
			if f := exp[0].evalI(c); f < 0 {
				crun.bindFacing = -1
			} else if f > 0 {
				crun.bindFacing = 1
			}
		case bindToParent_pos:
			x = exp[0].evalF(c) * redirscale
			if len(exp) > 1 {
				y = exp[1].evalF(c) * redirscale
				if len(exp) > 2 {
					z = exp[2].evalF(c) * redirscale
				}
			}
		case bindToParent_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})

	r := crun.root(true)
	if r == nil {
		return false
	}

	crun.bindPos[0] = x
	crun.bindPos[1] = y
	crun.bindPos[2] = z
	crun.setBindToId(r, false)
	crun.setBindTime(time)
	return false
}

type removeExplod StateControllerBase

const (
	removeExplod_id byte = iota
	removeExplod_redirectid
	removeExplod_index
)

func (sc removeExplod) Run(c *Char, _ []int32) bool {
	crun := c
	eid := int32(-1)
	idx := int32(-1)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case removeExplod_id:
			eid = exp[0].evalI(c)
		case removeExplod_index:
			idx = exp[0].evalI(c)
		case removeExplod_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.removeExplod(eid, idx)
	return false
}

type explodBindTime StateControllerBase

const (
	explodBindTime_id byte = iota
	explodBindTime_time
	explodBindTime_redirectid
)

func (sc explodBindTime) Run(c *Char, _ []int32) bool {
	crun := c
	var eid, time int32 = -1, 0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case explodBindTime_id:
			eid = exp[0].evalI(c)
		case explodBindTime_time:
			time = exp[0].evalI(c)
		case explodBindTime_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.explodBindTime(eid, time)
	return false
}

type moveHitReset StateControllerBase

const (
	moveHitReset_ byte = iota
	moveHitReset_redirectid
)

func (sc moveHitReset) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case moveHitReset_:
			crun.clearMoveHit()
		case moveHitReset_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type hitAdd StateControllerBase

const (
	hitAdd_value byte = iota
	hitAdd_redirectid
)

func (sc hitAdd) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case hitAdd_value:
			crun.hitAdd(exp[0].evalI(c))
		case hitAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type offset StateControllerBase

const (
	offset_x byte = iota
	offset_y
	offset_redirectid
)

func (sc offset) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case offset_x:
			crun.offset[0] = exp[0].evalF(c) * c.localscl
		case offset_y:
			crun.offset[1] = exp[0].evalF(c) * c.localscl
		case offset_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type victoryQuote StateControllerBase

const (
	victoryQuote_value byte = iota
	victoryQuote_redirectid
)

func (sc victoryQuote) Run(c *Char, _ []int32) bool {
	crun := c
	var v int32 = -1
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case victoryQuote_value:
			v = exp[0].evalI(c)
		case victoryQuote_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.winquote = v
	return false
}

type zoom StateControllerBase

const (
	zoom_pos byte = iota
	zoom_scale
	zoom_lag
	zoom_camerabound
	zoom_time
	zoom_stagebound
)

func (sc zoom) Run(c *Char, _ []int32) bool {
	pos := [2]float32{0, 0}
	t := int32(1)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case zoom_pos:
			pos[0] = exp[0].evalF(c) * c.localscl
			if len(exp) > 1 {
				pos[1] = exp[1].evalF(c) * c.localscl
			}
		case zoom_scale:
			sys.zoomScale = exp[0].evalF(c)
		case zoom_camerabound:
			sys.zoomCameraBound = exp[0].evalB(c)
		case zoom_stagebound:
			sys.zoomStageBound = exp[0].evalB(c)
		case zoom_lag:
			sys.zoomlag = exp[0].evalF(c)
		case zoom_time:
			t = exp[0].evalI(c)
		}
		return true
	})
	// This old calculation is both less accurate to Mugen and less intuitive to work with
	// sys.zoomPos[0] = sys.zoomScale * pos[0]
	sys.zoomPos[0] = pos[0]
	sys.zoomPos[1] = pos[1]
	sys.enableZoomtime = t
	return false
}

type forceFeedback StateControllerBase

const (
	forceFeedback_waveform byte = iota
	forceFeedback_time
	forceFeedback_freq
	forceFeedback_ampl
	forceFeedback_self
	forceFeedback_redirectid
)

func (sc forceFeedback) Run(c *Char, _ []int32) bool {
	/*crun := c
	waveform := int32(0)
	time := int32(60)
	freq := [4]float32{128, 0, 0, 0}
	ampl := [4]float32{128, 0, 0, 0}
	self := true
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case forceFeedback_waveform:
			waveform = exp[0].evalI(c)
		case forceFeedback_time:
			time = exp[0].evalI(c)
		case forceFeedback_freq:
			freq[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				freq[1] = exp[1].evalF(c)
			}
			if len(exp) > 2 {
				freq[2] = exp[2].evalF(c)
			}
			if len(exp) > 3 {
				freq[3] = exp[3].evalF(c)
			}
		case forceFeedback_ampl:
			ampl[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				ampl[1] = exp[1].evalF(c)
			}
			if len(exp) > 2 {
				ampl[2] = exp[2].evalF(c)
			}
			if len(exp) > 3 {
				ampl[3] = exp[3].evalF(c)
			}
		case forceFeedback_self:
			self = exp[0].evalB(c)
		case forceFeedback_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})*/
	// TODO: not implemented
	return false
}

type assertCommand StateControllerBase

const (
	assertCommand_name byte = iota
	assertCommand_buffertime
	assertCommand_redirectid
)

func (sc assertCommand) Run(c *Char, _ []int32) bool {
	crun := c
	n := ""
	bt := int32(1)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case assertCommand_name:
			n = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case assertCommand_buffertime:
			bt = exp[0].evalI(c)
		case assertCommand_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.assertCommand(n, bt)
	return false
}

type assertInput StateControllerBase

const (
	assertInput_flag byte = iota
	assertInput_redirectid
)

func (sc assertInput) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case assertInput_flag:
			crun.inputFlag |= InputBits(exp[0].evalI(c))
		case assertInput_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type dialogue StateControllerBase

const (
	dialogue_hidebars byte = iota
	dialogue_force
	dialogue_text
	dialogue_redirectid
)

func (sc dialogue) Run(c *Char, _ []int32) bool {
	crun := c
	reset := true
	force := false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case dialogue_hidebars:
			sys.dialogueBarsFlg = sys.lifebar.hidebars && exp[0].evalB(c)
		case dialogue_force:
			force = exp[0].evalB(c)
		case dialogue_text:
			sys.chars[crun.playerNo][0].appendDialogue(string(*(*[]byte)(unsafe.Pointer(&exp[0]))), reset)
			reset = false
		case dialogue_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if force {
		sys.dialogueFlg = true
		sys.dialogueForce = crun.playerNo + 1
	}
	return false
}

type dizzyPointsAdd StateControllerBase

const (
	dizzyPointsAdd_absolute byte = iota
	dizzyPointsAdd_value
	dizzyPointsAdd_redirectid
)

func (sc dizzyPointsAdd) Run(c *Char, _ []int32) bool {
	abs := false
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case dizzyPointsAdd_absolute:
			abs = exp[0].evalB(c)
		case dizzyPointsAdd_value:
			crun.dizzyPointsAdd(float64(exp[0].evalI(c)), abs)
		case dizzyPointsAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type dizzyPointsSet StateControllerBase

const (
	dizzyPointsSet_value byte = iota
	dizzyPointsSet_redirectid
)

func (sc dizzyPointsSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case dizzyPointsSet_value:
			crun.dizzyPointsSet(exp[0].evalI(c))
		case dizzyPointsSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type dizzySet StateControllerBase

const (
	dizzySet_value byte = iota
	dizzySet_redirectid
)

func (sc dizzySet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case dizzySet_value:
			crun.setDizzy(exp[0].evalB(c))
		case dizzySet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type guardBreakSet StateControllerBase

const (
	guardBreakSet_value byte = iota
	guardBreakSet_redirectid
)

func (sc guardBreakSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case guardBreakSet_value:
			crun.setGuardBreak(exp[0].evalB(c))
		case guardBreakSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type guardPointsAdd StateControllerBase

const (
	guardPointsAdd_absolute byte = iota
	guardPointsAdd_value
	guardPointsAdd_redirectid
)

func (sc guardPointsAdd) Run(c *Char, _ []int32) bool {
	abs := false
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case guardPointsAdd_absolute:
			abs = exp[0].evalB(c)
		case guardPointsAdd_value:
			crun.guardPointsAdd(float64(exp[0].evalI(c)), abs)
		case guardPointsAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type guardPointsSet StateControllerBase

const (
	guardPointsSet_value byte = iota
	guardPointsSet_redirectid
)

func (sc guardPointsSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case guardPointsSet_value:
			crun.guardPointsSet(exp[0].evalI(c))
		case guardPointsSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type lifebarAction StateControllerBase

const (
	lifebarAction_top byte = iota
	lifebarAction_time
	lifebarAction_timemul
	lifebarAction_anim
	lifebarAction_spr
	lifebarAction_snd
	lifebarAction_text
	lifebarAction_redirectid
)

func (sc lifebarAction) Run(c *Char, _ []int32) bool {
	crun := c
	var top bool
	var text string
	var timemul float32 = 1
	var time, anim int32 = -1, -1
	spr := [2]int32{-1, 0}
	snd := [2]int32{-1, 0}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case lifebarAction_top:
			top = exp[0].evalB(c)
		case lifebarAction_timemul:
			timemul = exp[0].evalF(c)
		case lifebarAction_time:
			time = exp[0].evalI(c)
		case lifebarAction_anim:
			anim = exp[0].evalI(c)
		case lifebarAction_spr:
			spr[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				spr[1] = exp[1].evalI(c)
			}
		case lifebarAction_snd:
			snd[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				snd[1] = exp[1].evalI(c)
			}
		case lifebarAction_text:
			text = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case lifebarAction_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.appendLifebarAction(text, snd, spr, anim, time, timemul, top)
	return false
}

type loadFile StateControllerBase

const (
	loadFile_path byte = iota
	loadFile_saveData
	loadFile_redirectid
)

func (sc loadFile) Run(c *Char, _ []int32) bool {
	crun := c
	var path string
	var data SaveData
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case loadFile_path:
			path = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case loadFile_saveData:
			data = SaveData(exp[0].evalI(c))
		case loadFile_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if path != "" {
		decodeFile, err := os.Open(filepath.Dir(c.gi().def) + "/" + path)
		if err != nil {
			defer decodeFile.Close()
			return false
		}
		defer decodeFile.Close()
		decoder := gob.NewDecoder(decodeFile)
		switch data {
		case SaveData_map:
			if err := decoder.Decode(&crun.mapArray); err != nil {
				panic(err)
			}
		case SaveData_var:
			if err := decoder.Decode(&crun.cnsvar); err != nil {
				panic(err)
			}
		case SaveData_fvar:
			if err := decoder.Decode(&crun.cnsfvar); err != nil {
				panic(err)
			}
		}
	}
	return false
}

type mapSet StateControllerBase

const (
	mapSet_mapArray byte = iota
	mapSet_value
	mapSet_redirectid
	mapSet_type
)

func (sc mapSet) Run(c *Char, _ []int32) bool {
	crun := c
	var s string
	var value float32
	var scType int32
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case mapSet_mapArray:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case mapSet_value:
			value = exp[0].evalF(c)
		case mapSet_type:
			scType = exp[0].evalI(c)
		case mapSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	crun.mapSet(s, value, scType)
	return false
}

type matchRestart StateControllerBase

const (
	matchRestart_reload byte = iota
	matchRestart_stagedef
	matchRestart_p1def
	matchRestart_p2def
	matchRestart_p3def
	matchRestart_p4def
	matchRestart_p5def
	matchRestart_p6def
	matchRestart_p7def
	matchRestart_p8def
)

func (sc matchRestart) Run(c *Char, _ []int32) bool {
	var s string
	reloadFlag := false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case matchRestart_reload:
			for i, p := range exp {
				sys.reloadCharSlot[i] = p.evalB(c)
				if sys.reloadCharSlot[i] {
					reloadFlag = true
				}
			}
		case matchRestart_stagedef:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.sdefOverwrite = SearchFile(s, []string{c.gi().def})
			//sys.reloadStageFlg = true
			reloadFlag = true
		case matchRestart_p1def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[0] = SearchFile(s, []string{c.gi().def})
		case matchRestart_p2def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[1] = SearchFile(s, []string{c.gi().def})
		case matchRestart_p3def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[2] = SearchFile(s, []string{c.gi().def})
		case matchRestart_p4def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[3] = SearchFile(s, []string{c.gi().def})
		case matchRestart_p5def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[4] = SearchFile(s, []string{c.gi().def})
		case matchRestart_p6def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[5] = SearchFile(s, []string{c.gi().def})
		case matchRestart_p7def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[6] = SearchFile(s, []string{c.gi().def})
		case matchRestart_p8def:
			s = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.sel.cdefOverwrite[7] = SearchFile(s, []string{c.gi().def})
		}
		return true
	})
	if sys.netConnection == nil && sys.replayFile == nil {
		if reloadFlag {
			sys.reloadFlg = true
		} else {
			sys.roundResetFlg = true
		}
	}
	return false
}

type printToConsole StateControllerBase

const (
	printToConsole_params byte = iota
	printToConsole_text
)

func (sc printToConsole) Run(c *Char, _ []int32) bool {
	params := []interface{}{}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case printToConsole_params:
			for _, e := range exp {
				if bv := e.run(c); bv.vtype == VT_Float {
					params = append(params, bv.ToF())
				} else {
					params = append(params, bv.ToI())
				}
			}
		case printToConsole_text:
			sys.printToConsole(sys.workingState.playerNo,
				int(exp[0].evalI(c)), params...)
		}
		return true
	})
	return false
}

type redLifeAdd StateControllerBase

const (
	redLifeAdd_absolute byte = iota
	redLifeAdd_value
	redLifeAdd_redirectid
)

func (sc redLifeAdd) Run(c *Char, _ []int32) bool {
	abs := false
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case redLifeAdd_absolute:
			abs = exp[0].evalB(c)
		case redLifeAdd_value:
			crun.redLifeAdd(float64(exp[0].evalI(c)), abs)
		case redLifeAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type redLifeSet StateControllerBase

const (
	redLifeSet_value byte = iota
	redLifeSet_redirectid
)

func (sc redLifeSet) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case redLifeSet_value:
			crun.redLifeSet(exp[0].evalI(c))
		case redLifeSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type remapSprite StateControllerBase

const (
	remapSprite_reset byte = iota
	remapSprite_preset
	remapSprite_source
	remapSprite_dest
	remapSprite_redirectid
)

func (sc remapSprite) Run(c *Char, _ []int32) bool {
	crun := c
	src := [...]int16{-1, -1}

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case remapSprite_reset:
			if exp[0].evalB(c) {
				crun.remapSpr = make(RemapPreset)
			}
		case remapSprite_preset:
			crun.remapSpritePreset(string(*(*[]byte)(unsafe.Pointer(&exp[0]))))
		case remapSprite_source:
			src[0] = int16(exp[0].evalI(c))
			if len(exp) > 1 {
				src[1] = int16(exp[1].evalI(c))
			}
		case remapSprite_dest:
			dst := [...]int16{int16(exp[0].evalI(c)), -1}
			if len(exp) > 1 {
				dst[1] = int16(exp[1].evalI(c))
			}
			crun.remapSprite(src, dst)
		case remapSprite_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})

	crun.anim.remap = crun.remapSpr

	// Update sprite in case current sprite was remapped
	// https://github.com/ikemen-engine/Ikemen-GO/issues/2456
	if crun.anim != nil {
		crun.anim.newframe = true
		crun.anim.UpdateSprite()
	}

	return false
}

type roundTimeAdd StateControllerBase

const (
	roundTimeAdd_value byte = iota
	roundTimeAdd_redirectid
)

func (sc roundTimeAdd) Run(c *Char, _ []int32) bool {
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case roundTimeAdd_value:
			if sys.roundTime != -1 {
				sys.time = Clamp(sys.time+exp[0].evalI(c), 0, sys.roundTime)
			}
		}
		return true
	})
	return false
}

type roundTimeSet StateControllerBase

const (
	roundTimeSet_value byte = iota
	roundTimeSet_redirectid
)

func (sc roundTimeSet) Run(c *Char, _ []int32) bool {
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case roundTimeSet_value:
			if sys.roundTime != -1 {
				sys.time = Clamp(exp[0].evalI(c), 0, sys.roundTime)
			}
		}
		return true
	})
	return false
}

type saveFile StateControllerBase

const (
	saveFile_path byte = iota
	saveFile_saveData
	saveFile_redirectid
)

func (sc saveFile) Run(c *Char, _ []int32) bool {
	crun := c
	var path string
	var data SaveData
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case saveFile_path:
			path = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case saveFile_saveData:
			data = SaveData(exp[0].evalI(c))
		case saveFile_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if path != "" {
		encodeFile, err := os.Create(filepath.Dir(c.gi().def) + "/" + path)
		if err != nil {
			panic(err)
		}
		defer encodeFile.Close()
		encoder := gob.NewEncoder(encodeFile)
		switch data {
		case SaveData_map:
			if err := encoder.Encode(crun.mapArray); err != nil {
				panic(err)
			}
		case SaveData_var:
			if err := encoder.Encode(crun.cnsvar); err != nil {
				panic(err)
			}
		case SaveData_fvar:
			if err := encoder.Encode(crun.cnsfvar); err != nil {
				panic(err)
			}
		}
	}
	return false
}

type scoreAdd StateControllerBase

const (
	scoreAdd_value byte = iota
	scoreAdd_redirectid
)

func (sc scoreAdd) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case scoreAdd_value:
			crun.scoreAdd(exp[0].evalF(c))
		case scoreAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type modifyBGCtrl StateControllerBase

const (
	modifyBGCtrl_id byte = iota
	modifyBGCtrl_time
	modifyBGCtrl_value
	modifyBGCtrl_x
	modifyBGCtrl_y
	modifyBGCtrl_source
	modifyBGCtrl_dest
	modifyBGCtrl_add
	modifyBGCtrl_mul
	modifyBGCtrl_sinadd
	modifyBGCtrl_sinmul
	modifyBGCtrl_sincolor
	modifyBGCtrl_sinhue
	modifyBGCtrl_invertall
	modifyBGCtrl_invertblend
	modifyBGCtrl_color
	modifyBGCtrl_hue
	modifyBGCtrl_redirectid
)

func (sc modifyBGCtrl) Run(c *Char, _ []int32) bool {
	//crun := c
	var cid int32
	t, v := [3]int32{IErr, IErr, IErr}, [3]int32{IErr, IErr, IErr}
	x, y := float32(math.NaN()), float32(math.NaN())
	src, dst := [2]int32{IErr, IErr}, [2]int32{IErr, IErr}
	add, mul := [3]int32{IErr, IErr, IErr}, [3]int32{IErr, IErr, IErr}
	sinadd, sinmul := [4]int32{IErr, IErr, IErr, IErr}, [4]int32{IErr, IErr, IErr, IErr}
	sincolor, sinhue := [2]int32{IErr, IErr}, [2]int32{IErr, IErr}
	invall, invblend, color, hue := IErr, IErr, float32(math.NaN()), float32(math.NaN())
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyBGCtrl_id:
			cid = exp[0].evalI(c)
		case modifyBGCtrl_time:
			t[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				t[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					t[2] = exp[2].evalI(c)
				}
			}
		case modifyBGCtrl_value:
			v[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				v[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					v[2] = exp[2].evalI(c)
				}
			}
		case modifyBGCtrl_x:
			x = exp[0].evalF(c)
		case modifyBGCtrl_y:
			y = exp[0].evalF(c)
		case modifyBGCtrl_source:
			src[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				src[1] = exp[1].evalI(c)
			}
		case modifyBGCtrl_dest:
			dst[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				dst[1] = exp[1].evalI(c)
			}
		case modifyBGCtrl_add:
			add[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				add[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					add[2] = exp[2].evalI(c)
				}
			}
		case modifyBGCtrl_mul:
			mul[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				mul[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					mul[2] = exp[2].evalI(c)
				}
			}
		case modifyBGCtrl_sinadd:
			sinadd[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				sinadd[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					sinadd[2] = exp[2].evalI(c)
					if len(exp) > 3 {
						sinadd[3] = exp[3].evalI(c)
					}
				}
			}
		case modifyBGCtrl_sinmul:
			sinmul[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				sinmul[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					sinmul[2] = exp[2].evalI(c)
					if len(exp) > 3 {
						sinmul[3] = exp[3].evalI(c)
					}
				}
			}
		case modifyBGCtrl_sincolor:
			sincolor[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				sincolor[1] = exp[1].evalI(c)
			}
		case modifyBGCtrl_sinhue:
			sinhue[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				sinhue[1] = exp[1].evalI(c)
			}
		case modifyBGCtrl_invertall:
			invall = exp[0].evalI(c)
		case modifyBGCtrl_invertblend:
			invblend = exp[0].evalI(c)
		case modifyBGCtrl_color:
			color = exp[0].evalF(c)
		case modifyBGCtrl_hue:
			hue = exp[0].evalF(c)
		case modifyBGCtrl_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				//crun = rid
			} else {
				return false
			}
		}
		return true
	})
	sys.stage.modifyBGCtrl(cid, t, v, x, y, src, dst, add, mul, sinadd, sinmul, sincolor, sinhue, invall, invblend, color, hue)
	return false
}

type modifyBGCtrl3d StateControllerBase

const (
	modifyBGCtrl3d_ctrlid byte = iota
	modifyBGCtrl3d_time
	modifyBGCtrl3d_value
	modifyBGCtrl3d_redirectid
)

func (sc modifyBGCtrl3d) Run(c *Char, _ []int32) bool {
	//crun := c

	var cid int32
	t, v := [3]int32{IErr, IErr, IErr}, [3]int32{IErr, IErr, IErr}
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyBGCtrl3d_ctrlid:
			cid = exp[0].evalI(c)
		case modifyBGCtrl3d_time:
			t[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				t[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					t[2] = exp[2].evalI(c)
				}
			}
		case modifyBGCtrl3d_value:
			v[0] = exp[0].evalI(c)
			if len(exp) > 1 {
				v[1] = exp[1].evalI(c)
				if len(exp) > 2 {
					v[2] = exp[2].evalI(c)
				}
			}
		case modifyBGCtrl3d_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				//crun = rid
			} else {
				return false
			}
		}
		return true
	})
	sys.stage.modifyBGCtrl3d(uint32(cid), t, v)
	return false
}

type modifyBgm StateControllerBase

const (
	modifyBgm_volume = iota
	modifyBgm_loopstart
	modifyBgm_loopend
	modifyBgm_position
	modifyBgm_freqmul
	modifyBgm_redirectid
)

func (sc modifyBgm) Run(c *Char, _ []int32) bool {
	var volumeSet, loopStartSet, loopEndSet, posSet, freqSet = false, false, false, false, false
	var volume, loopstart, loopend, position int = 100, 0, 0, 0
	var freqmul float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyBgm_volume:
			volume = int(exp[0].evalI(c))
			volumeSet = true
		case modifyBgm_loopstart:
			loopstart = int(exp[0].evalI64(c))
			loopStartSet = true
		case modifyBgm_loopend:
			loopend = int(exp[0].evalI64(c))
			loopEndSet = true
		case modifyBgm_position:
			position = int(exp[0].evalI64(c))
			posSet = true
		case modifyBgm_freqmul:
			freqmul = float32(exp[0].evalF(c))
			freqSet = true
		case modifyBgm_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {

			} else {
				return false
			}
		}
		return true
	})
	if sys.bgm.ctrl != nil {
		// Set values that are different only
		if volumeSet {
			volumeScaled := int(float64(volume) / 100.0 * float64(sys.cfg.Sound.MaxBGMVolume))
			sys.bgm.bgmVolume = int(Min(int32(volumeScaled), int32(sys.cfg.Sound.MaxBGMVolume)))
			sys.bgm.UpdateVolume()
		}
		if posSet {
			sys.bgm.Seek(position)
		}
		if sl, ok := sys.bgm.volctrl.Streamer.(*StreamLooper); ok {
			if (loopStartSet && sl.loopstart != loopstart) || (loopEndSet && sl.loopend != loopend) {
				sys.bgm.SetLoopPoints(loopstart, loopend)
			}
		}
		if freqSet && sys.bgm.freqmul != freqmul {
			sys.bgm.SetFreqMul(freqmul)
		}
	}
	return false
}

type modifySnd StateControllerBase

const (
	modifySnd_channel = iota
	modifySnd_pan
	modifySnd_abspan
	modifySnd_volume
	modifySnd_volumescale
	modifySnd_freqmul
	modifySnd_redirectid
	modifySnd_priority
	modifySnd_loopstart
	modifySnd_loopend
	modifySnd_position
	modifySnd_loop
	modifySnd_loopcount
	modifySnd_stopongethit
	modifySnd_stoponchangestate
)

func (sc modifySnd) Run(c *Char, _ []int32) bool {
	if sys.noSoundFlg {
		return false
	}
	crun := c
	snd := crun.soundChannels.Get(-1)
	var ch, pri int32 = -1, 0
	var vo, fr float32 = 100, 1.0
	stopgh, stopcs := false, false
	freqMulSet, volumeSet, prioritySet, panSet, loopStartSet, loopEndSet, posSet, lcSet, loopSet := false, false, false, false, false, false, false, false, false
	stopghSet, stopcsSet := false, false
	var loopstart, loopend, position, lc int = 0, 0, 0, 0
	var p float32 = 0
	x := &c.pos[0]
	ls := crun.localscl
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifySnd_channel:
			ch = exp[0].evalI(c)
		case modifySnd_pan:
			p = exp[0].evalF(c)
			panSet = true
		case modifySnd_abspan:
			x = nil
			ls = 1
			p = exp[0].evalF(c)
			panSet = true
		case modifySnd_volume:
			vo = (vo + float32(exp[0].evalI(c))*(25.0/64.0)) * (64.0 / 25.0)
			volumeSet = true
		case modifySnd_volumescale:
			vo = float32(crun.gi().data.volume * exp[0].evalI(c) / 100)
			volumeSet = true
		case modifySnd_freqmul:
			fr = ClampF(exp[0].evalF(c), 0.01, 5)
			freqMulSet = true
		case modifySnd_priority:
			pri = exp[0].evalI(c)
			prioritySet = true
		case modifySnd_loopstart:
			loopstart = int(exp[0].evalI64(c))
			loopStartSet = true
		case modifySnd_loopend:
			loopend = int(exp[0].evalI64(c))
			loopEndSet = true
		case modifySnd_position:
			position = int(exp[0].evalI64(c))
			posSet = true
		case modifySnd_loop:
			if lc == 0 {
				if bool(exp[0].evalB(c)) {
					lc = -1
				} else {
					lc = 0
				}
				loopSet = true
			}
		case modifySnd_loopcount:
			tmp := int(exp[0].evalI(c))
			if tmp < 0 {
				lc = -1
			} else {
				lc = MaxI(tmp-1, 0)
			}
			lcSet = true
		case modifySnd_stopongethit:
			stopgh = exp[0].evalB(c)
		case modifySnd_stoponchangestate:
			stopcs = exp[0].evalB(c)
		case modifySnd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				x = &crun.pos[0]
				ls = crun.localscl
				snd = crun.soundChannels.Get(ch)
			} else {
				return false
			}
		}
		return true
	})
	// Grab the correct sound channel now
	channelCount := 1
	if ch < 0 {
		channelCount = len(crun.soundChannels.channels)
	}
	for i := channelCount - 1; i >= 0; i-- {
		if ch < 0 {
			snd = &crun.soundChannels.channels[i]
		} else {
			snd = crun.soundChannels.Get(ch)
		}

		if snd != nil && snd.sfx != nil {
			// If we didn't set the values, default them to current values.
			if !freqMulSet {
				fr = snd.sfx.freqmul
			}
			if !volumeSet {
				vo = snd.sfx.volume
			}
			if !prioritySet {
				pri = snd.sfx.priority
			}
			if !panSet {
				p = snd.sfx.p
				ls = snd.sfx.ls
				x = snd.sfx.x
			}

			// Now set the values if they're different
			if snd.sfx.freqmul != fr {
				snd.SetFreqMul(fr)
			}
			if pri != snd.sfx.priority {
				snd.SetPriority(pri)
			}
			if posSet {
				snd.streamer.Seek(position)
			}
			if lcSet || loopSet {
				if sl, ok := snd.sfx.streamer.(*StreamLooper); ok {
					sl.loopcount = lc
				}
			}
			if sl, ok := snd.sfx.streamer.(*StreamLooper); ok {
				if (loopStartSet && sl.loopstart != loopstart) || (loopEndSet && sl.loopend != loopend) {
					snd.SetLoopPoints(loopstart, loopend)
				}
			}
			if p != snd.sfx.p || ls != snd.sfx.ls || x != snd.sfx.x {
				snd.SetPan(p*crun.facing, ls, x)
			}
			if vo != snd.sfx.volume {
				snd.SetVolume(vo)
			}
			// These flags can be updated regardless since there are no calculations involved
			if stopghSet {
				snd.stopOnGetHit = stopgh
			}
			if stopcsSet {
				snd.stopOnChangeState = stopcs
			}
		}
	}
	return false
}

type playBgm StateControllerBase

const (
	playBgm_bgm = iota
	playBgm_volume
	playBgm_loop
	playBgm_loopstart
	playBgm_loopend
	playBgm_startposition
	playBgm_freqmul
	playBgm_loopcount
	playBgm_redirectid
)

func (sc playBgm) Run(c *Char, _ []int32) bool {
	crun := c
	var b, totalRecall bool
	var bgm string
	var loop, loopcount, volume, loopstart, loopend, startposition int = 1, -1, 100, 0, 0, 0
	var freqmul float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case playBgm_bgm:
			bgm = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			// Default to stage BGM if string is stage
			if bgm == "stage" {
				// Search .def directory last in this instance
				bgm = SearchFile(sys.stage.bgmusic, []string{sys.stage.def, "", "sound/", crun.gi().def})
				totalRecall = true
			} else if bgm != "" {
				bgm = SearchFile(bgm, []string{crun.gi().def, sys.stage.def, "", "sound/"})
			}
			b = true
		case playBgm_volume:
			volume = int(exp[0].evalI(c))
			if !b {
				sys.bgm.bgmVolume = int(Min(int32(volume), int32(sys.cfg.Sound.MaxBGMVolume)))
				sys.bgm.UpdateVolume()
			}
		case playBgm_loop:
			loop = int(exp[0].evalI(c))
		case playBgm_loopstart:
			loopstart = int(exp[0].evalI(c))
		case playBgm_loopend:
			loopend = int(exp[0].evalI(c))
		case playBgm_startposition:
			startposition = int(exp[0].evalI(c))
		case playBgm_freqmul:
			freqmul = exp[0].evalF(c)
		case playBgm_loopcount:
			loopcount = int(exp[0].evalI(c))
		case playBgm_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	if b {
		// Recall all the stage info
		if totalRecall {
			volume = int(sys.stage.bgmvolume)
			startposition = int(sys.stage.bgmstartposition)
			loopstart = int(sys.stage.bgmloopstart)
			loopend = int(sys.stage.bgmloopend)
			freqmul = sys.stage.bgmfreqmul
		}
		sys.bgm.Open(bgm, loop, volume, loopstart, loopend, startposition, freqmul, loopcount)
		sys.playBgmFlg = true
	}
	return false
}

type targetDizzyPointsAdd StateControllerBase

const (
	targetDizzyPointsAdd_id byte = iota
	targetDizzyPointsAdd_index
	targetDizzyPointsAdd_absolute
	targetDizzyPointsAdd_value
	targetDizzyPointsAdd_redirectid
)

func (sc targetDizzyPointsAdd) Run(c *Char, _ []int32) bool {
	crun := c
	abs := false
	tid, tidx := int32(-1), int(-1)
	vl := int32(0)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetDizzyPointsAdd_id:
			tid = exp[0].evalI(c)
		case targetDizzyPointsAdd_index:
			tidx = int(exp[0].evalI(c))
		case targetDizzyPointsAdd_absolute:
			abs = exp[0].evalB(c)
		case targetDizzyPointsAdd_value:
			vl = exp[0].evalI(c)
		case targetDizzyPointsAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetDizzyPointsAdd(tar, vl, abs)
	}
	return false
}

type targetGuardPointsAdd StateControllerBase

const (
	targetGuardPointsAdd_id byte = iota
	targetGuardPointsAdd_index
	targetGuardPointsAdd_absolute
	targetGuardPointsAdd_value
	targetGuardPointsAdd_redirectid
)

func (sc targetGuardPointsAdd) Run(c *Char, _ []int32) bool {
	crun := c
	abs := false
	tid, tidx := int32(-1), int(-1)
	vl := int32(0)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetGuardPointsAdd_id:
			tid = exp[0].evalI(c)
		case targetGuardPointsAdd_index:
			tidx = int(exp[0].evalI(c))
		case targetGuardPointsAdd_absolute:
			abs = exp[0].evalB(c)
		case targetGuardPointsAdd_value:
			vl = exp[0].evalI(c)
		case targetGuardPointsAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetGuardPointsAdd(tar, vl, abs)
	}
	return false
}

type targetRedLifeAdd StateControllerBase

const (
	targetRedLifeAdd_id byte = iota
	targetRedLifeAdd_index
	targetRedLifeAdd_absolute
	targetRedLifeAdd_value
	targetRedLifeAdd_redirectid
)

func (sc targetRedLifeAdd) Run(c *Char, _ []int32) bool {
	crun := c
	abs := false
	tid, tidx := int32(-1), int(-1)
	vl := int32(0)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetRedLifeAdd_id:
			tid = exp[0].evalI(c)
		case targetRedLifeAdd_index:
			tidx = int(exp[0].evalI(c))
		case targetRedLifeAdd_absolute:
			abs = exp[0].evalB(c)
		case targetRedLifeAdd_value:
			vl = exp[0].evalI(c)
		case targetRedLifeAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	// Mugen forces absolute parameter when healing characters
	if vl > 0 && c.stWgi().ikemenver[0] == 0 && c.stWgi().ikemenver[1] == 0 {
		abs = true
	}
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetRedLifeAdd(tar, vl, abs)
	}
	return false
}

type targetScoreAdd StateControllerBase

const (
	targetScoreAdd_id byte = iota
	targetScoreAdd_index
	targetScoreAdd_value
	targetScoreAdd_redirectid
)

func (sc targetScoreAdd) Run(c *Char, _ []int32) bool {
	crun := c
	tid, tidx := int32(-1), int(-1)
	vl := float32(0)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetScoreAdd_id:
			tid = exp[0].evalI(c)
		case targetScoreAdd_index:
			tidx = int(exp[0].evalI(c))
		case targetScoreAdd_value:
			vl = exp[0].evalF(c)
		case targetScoreAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	tar := crun.getTarget(tid, tidx)
	if len(tar) > 0 {
		crun.targetScoreAdd(tar, vl)
	}
	return false
}

type text StateControllerBase

const (
	text_removetime byte = iota + palFX_last + 1
	text_layerno
	text_params
	text_font
	text_localcoord
	text_bank
	text_align
	text_linespacing
	text_textdelay
	text_text
	text_pos
	text_velocity
	text_friction
	text_accel
	text_scale
	text_color
	text_xshear
	text_id
	text_redirectid
	text_last = iota + palFX_last + 1 - 1
)

func (sc text) Run(c *Char, _ []int32) bool {
	crun := c
	params := []interface{}{}
	ownerID := crun.id
	ts := NewTextSprite()
	ts.SetLocalcoord(float32(sys.scrrect[2]), float32(sys.scrrect[3]))
	var xscl, yscl float32 = 1, 1
	var fnt int = -1
	ts.ownerid = ownerID
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case text_removetime:
			ts.removetime = exp[0].evalI(c)
		case text_layerno:
			ts.layerno = int16(exp[0].evalI(c))
		case text_params:
			for _, e := range exp {
				if bv := e.run(c); bv.vtype == VT_Float {
					params = append(params, bv.ToF())
				} else {
					params = append(params, bv.ToI())
				}
			}
		case text_text:
			sn := int(exp[0].evalI(c))
			spl := sys.stringPool[sys.workingState.playerNo].List
			if sn >= 0 && sn < len(spl) {
				ts.text = OldSprintf(spl[sn], params...)
			}
		case text_font:
			fnt = int(exp[1].evalI(c))
			fflg := exp[0].evalB(c)
			fntList := crun.gi().fnt
			if fflg {
				fntList = sys.lifebar.fnt
			}
			if fnt >= 0 && fnt < len(fntList) && fntList[fnt] != nil {
				ts.fnt = fntList[fnt]
				if fflg {
					ts.SetLocalcoord(float32(sys.lifebarLocalcoord[0]), float32(sys.lifebarLocalcoord[1]))
				} else {
					//ts.SetLocalcoord(c.stOgi().localcoord[0], c.stOgi().localcoord[1])
				}
			} else {
				fnt = -1
			}
		case text_localcoord:
			ts.SetLocalcoord(exp[0].evalF(c), exp[1].evalF(c))
		case text_bank:
			ts.bank = exp[0].evalI(c)
		case text_align:
			ts.align = exp[0].evalI(c)
		case text_linespacing:
			ts.lineSpacing = exp[0].evalF(c)
		case text_textdelay:
			ts.textDelay = exp[0].evalF(c)
		case text_pos:
			ts.x = exp[0].evalF(c)/ts.localScale + float32(ts.offsetX)
			if len(exp) > 1 {
				ts.y = exp[1].evalF(c) / ts.localScale
			}
		case text_velocity:
			ts.velocity[0] = exp[0].evalF(c) / ts.localScale
			if len(exp) > 1 {
				ts.velocity[1] = exp[1].evalF(c) / ts.localScale
			}
		case text_friction:
			ts.friction[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				ts.friction[1] = exp[1].evalF(c)
			}
		case text_accel:
			ts.accel[0] = exp[0].evalF(c) / ts.localScale
			if len(exp) > 1 {
				ts.accel[1] = exp[1].evalF(c) / ts.localScale
			}
		case text_scale:
			xscl = exp[0].evalF(c)
			if len(exp) > 1 {
				yscl = exp[1].evalF(c)
			}
		case text_color:
			var r, g, b int32 = exp[0].evalI(c), 255, 255
			if len(exp) > 1 {
				g = exp[1].evalI(c)
				if len(exp) > 2 {
					b = exp[2].evalI(c)
				}
			}
			ts.SetColor(r, g, b)
		case text_xshear:
			ts.xshear = exp[0].evalF(c)
		case text_id:
			ts.id = exp[0].evalI(c)
		case text_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				ts.ownerid = crun.id
			} else {
				return false
			}
		default:
			if applyTextPalFX(ts, paramID, exp, c) {
				break
			}
		}
		return true
	})
	ts.xscl = xscl / ts.localScale
	ts.yscl = yscl / ts.localScale
	if fnt == -1 {
		ts.fnt = sys.debugFont.fnt
		ts.xscl *= sys.debugFont.xscl
		ts.yscl *= sys.debugFont.yscl
	}
	if ts.text == "" {
		ts.text = OldSprintf("%v", params...)
	}
	sys.lifebar.textsprite = append(sys.lifebar.textsprite, ts)
	return false
}

func applyTextPalFX(ts *TextSprite, paramID byte, exp []BytecodeExp, c *Char) bool {
	switch paramID {
	case palFX_time:
		ts.palfx.time = exp[0].evalI(c) * 2
	case palFX_color:
		ts.palfx.color = exp[0].evalF(c) / 256
	case palFX_hue:
		ts.palfx.hue = exp[0].evalF(c) / 256
	case palFX_add:
		ts.palfx.add[0] = exp[0].evalI(c)
		ts.palfx.add[1] = exp[1].evalI(c)
		ts.palfx.add[2] = exp[2].evalI(c)
	case palFX_mul:
		ts.palfx.mul[0] = exp[0].evalI(c)
		ts.palfx.mul[1] = exp[1].evalI(c)
		ts.palfx.mul[2] = exp[2].evalI(c)
	case palFX_sinadd:
		var side int32 = 1
		if len(exp) > 3 {
			if exp[3].evalI(c) < 0 {
				ts.palfx.cycletime[0] = -exp[3].evalI(c) * 2
				side = -1
			} else {
				ts.palfx.cycletime[0] = exp[3].evalI(c) * 2
			}
		}
		ts.palfx.sinadd[0] = exp[0].evalI(c) * side
		ts.palfx.sinadd[1] = exp[1].evalI(c) * side
		ts.palfx.sinadd[2] = exp[2].evalI(c) * side
	case palFX_sinmul:
		var side int32 = 1
		if len(exp) > 3 {
			if exp[3].evalI(c) < 0 {
				ts.palfx.cycletime[1] = -exp[3].evalI(c) * 2
				side = -1
			} else {
				ts.palfx.cycletime[1] = exp[3].evalI(c) * 2
			}
		}
		ts.palfx.sinmul[0] = exp[0].evalI(c) * side
		ts.palfx.sinmul[1] = exp[1].evalI(c) * side
		ts.palfx.sinmul[2] = exp[2].evalI(c) * side
	case palFX_sincolor:
		var side int32 = 1
		if len(exp) > 1 {
			if exp[1].evalI(c) < 0 {
				ts.palfx.cycletime[2] = -exp[1].evalI(c) * 2
				side = -1
			} else {
				ts.palfx.cycletime[2] = exp[1].evalI(c) * 2
			}
		}
		ts.palfx.sincolor = exp[0].evalI(c) * side
	case palFX_sinhue:
		var side int32 = 1
		if len(exp) > 1 {
			if exp[1].evalI(c) < 0 {
				ts.palfx.cycletime[3] = -exp[1].evalI(c) * 2
				side = -1
			} else {
				ts.palfx.cycletime[3] = exp[1].evalI(c) * 2
			}
		}
		ts.palfx.sinhue = exp[0].evalI(c) * side
	case palFX_invertall:
		ts.palfx.invertall = exp[0].evalB(c)
	case palFX_invertblend:
		ts.palfx.invertblend = Clamp(exp[0].evalI(c), -1, 2)
	default:
		return false
	}
	return false
}

type removeText StateControllerBase

const (
	removetext_id byte = iota
	removetext_redirectid
)

func (sc removeText) Run(c *Char, _ []int32) bool {
	crun := c
	textID := int32(-1)
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case removetext_id:
			textID = exp[0].evalI(c)
		case removetext_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	sys.lifebar.RemoveText(textID, crun.id)
	return false
}

// Platform bytecode definitons
type createPlatform StateControllerBase

const (
	createPlatform_id byte = iota
	createPlatform_name
	createPlatform_anim
	createPlatform_pos
	createPlatform_size
	createPlatform_offset
	createPlatform_activeTime
	createPlatform_redirectid
)

// The createPlatform bytecode function.
func (sc createPlatform) Run(schara *Char, _ []int32) bool {
	var chara = schara
	var customOffset = false
	var plat = Platform{
		anim:       -1,
		pos:        [2]float32{0, 0},
		size:       [2]int32{0, 0},
		offset:     [2]int32{0, 0},
		activeTime: -1,
	}

	StateControllerBase(sc).run(schara, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case createPlatform_id:
			plat.id = exp[0].evalI(schara)
		case createPlatform_name:
			plat.name = string(*(*[]byte)(unsafe.Pointer(&exp[0])))
		case createPlatform_pos:
			plat.pos[0] = exp[0].evalF(schara)
			plat.pos[1] = exp[1].evalF(schara)
		case createPlatform_size:
			plat.size[0] = exp[0].evalI(schara)
			plat.size[1] = exp[1].evalI(schara)
		case createPlatform_offset:
			customOffset = true
			plat.offset[0] = exp[0].evalI(schara)
			plat.offset[1] = exp[1].evalI(schara)
		case createPlatform_activeTime:
			plat.activeTime = exp[0].evalI(schara)
		case createPlatform_redirectid:
			if rid := sys.playerID(exp[0].evalI(schara)); rid != nil {
				chara = rid
			} else {
				return false
			}
		}
		return true
	})

	if !customOffset {
		if plat.size[0] != 0 {
			plat.offset[0] = plat.size[0] / 2
		}
		if plat.size[1] != 0 {
			plat.offset[1] = plat.size[1] / 2
		}
	}
	plat.ownerID = chara.id

	return false
}

type removePlatform StateControllerBase

const (
	removePlatform_id byte = iota
	removePlatform_name
)

type modifyStageVar StateControllerBase

const (
	modifyStageVar_camera_boundleft byte = iota
	modifyStageVar_camera_boundright
	modifyStageVar_camera_boundhigh
	modifyStageVar_camera_boundlow
	modifyStageVar_camera_verticalfollow
	modifyStageVar_camera_floortension
	modifyStageVar_camera_tensionhigh
	modifyStageVar_camera_tensionlow
	modifyStageVar_camera_tension
	modifyStageVar_camera_tensionvel
	modifyStageVar_camera_cuthigh
	modifyStageVar_camera_cutlow
	modifyStageVar_camera_startzoom
	modifyStageVar_camera_zoomout
	modifyStageVar_camera_zoomin
	modifyStageVar_camera_zoomindelay
	modifyStageVar_camera_zoominspeed
	modifyStageVar_camera_zoomoutspeed
	modifyStageVar_camera_yscrollspeed
	modifyStageVar_camera_ytension_enable
	modifyStageVar_camera_autocenter
	modifyStageVar_camera_lowestcap
	modifyStageVar_playerinfo_leftbound
	modifyStageVar_playerinfo_rightbound
	modifyStageVar_playerinfo_topbound
	modifyStageVar_playerinfo_botbound
	modifyStageVar_scaling_topz
	modifyStageVar_scaling_botz
	modifyStageVar_scaling_topscale
	modifyStageVar_scaling_botscale
	modifyStageVar_bound_screenleft
	modifyStageVar_bound_screenright
	modifyStageVar_stageinfo_zoffset
	modifyStageVar_stageinfo_zoffsetlink
	modifyStageVar_stageinfo_xscale
	modifyStageVar_stageinfo_yscale
	modifyStageVar_shadow_intensity
	modifyStageVar_shadow_color
	modifyStageVar_shadow_yscale
	modifyStageVar_shadow_fade_range
	modifyStageVar_shadow_xshear
	modifyStageVar_shadow_offset
	modifyStageVar_shadow_window
	modifyStageVar_reflection_intensity
	modifyStageVar_reflection_yscale
	modifyStageVar_reflection_xshear
	modifyStageVar_reflection_color
	modifyStageVar_reflection_offset
	modifyStageVar_reflection_window
)

func (sc modifyStageVar) Run(c *Char, _ []int32) bool {
	//crun := c RedirectID is pointless when modifying a stage
	s := sys.stage
	shouldResetCamera := false
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		// Camera group
		case modifyStageVar_camera_autocenter:
			s.stageCamera.autocenter = exp[0].evalB(c)
			shouldResetCamera = true
		case modifyStageVar_camera_boundleft:
			s.stageCamera.boundleft = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_boundright:
			s.stageCamera.boundright = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_boundhigh:
			s.stageCamera.boundhigh = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_boundlow:
			s.stageCamera.boundlow = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_verticalfollow:
			s.stageCamera.verticalfollow = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_floortension:
			s.stageCamera.floortension = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_lowestcap:
			s.stageCamera.lowestcap = exp[0].evalB(c)
			shouldResetCamera = true
		case modifyStageVar_camera_tensionhigh:
			s.stageCamera.tensionhigh = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_tensionlow:
			s.stageCamera.tensionlow = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_tension:
			s.stageCamera.tension = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_tensionvel:
			s.stageCamera.tensionvel = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_cuthigh:
			s.stageCamera.cuthigh = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_cutlow:
			s.stageCamera.cutlow = int32(exp[0].evalF(c) * c.localscl / s.localscl)
			shouldResetCamera = true
		case modifyStageVar_camera_startzoom:
			s.stageCamera.startzoom = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_zoomout:
			s.stageCamera.zoomout = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_zoomin:
			s.stageCamera.zoomin = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_zoomindelay:
			s.stageCamera.zoomindelay = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_zoominspeed:
			s.stageCamera.zoominspeed = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_zoomoutspeed:
			s.stageCamera.zoomoutspeed = exp[0].evalF(c)
			shouldResetCamera = true
		case modifyStageVar_camera_ytension_enable:
			s.stageCamera.ytensionenable = exp[0].evalB(c)
			shouldResetCamera = true
		case modifyStageVar_camera_yscrollspeed:
			s.stageCamera.yscrollspeed = exp[0].evalF(c)
			shouldResetCamera = true
		// PlayerInfo group
		case modifyStageVar_playerinfo_leftbound:
			s.leftbound = exp[0].evalF(c) * c.localscl / s.localscl
		case modifyStageVar_playerinfo_rightbound:
			s.rightbound = exp[0].evalF(c) * c.localscl / s.localscl
		case modifyStageVar_playerinfo_topbound:
			s.topbound = exp[0].evalF(c) * c.localscl / s.localscl
		case modifyStageVar_playerinfo_botbound:
			s.botbound = exp[0].evalF(c) * c.localscl / s.localscl
		// Scaling group
		case modifyStageVar_scaling_topz:
			if s.mugenver[0] != 1 { // mugen 1.0+ removed support for topz
				s.stageCamera.topz = exp[0].evalF(c)
			}
		case modifyStageVar_scaling_botz:
			if s.mugenver[0] != 1 { // mugen 1.0+ removed support for botz
				s.stageCamera.botz = exp[0].evalF(c)
			}
		case modifyStageVar_scaling_topscale:
			if s.mugenver[0] != 1 { // mugen 1.0+ removed support for topscale
				s.stageCamera.ztopscale = exp[0].evalF(c)
			}
		case modifyStageVar_scaling_botscale:
			if s.mugenver[0] != 1 { // mugen 1.0+ removed support for botscale
				s.stageCamera.zbotscale = exp[0].evalF(c)
			}
		// Bound group
		case modifyStageVar_bound_screenleft:
			s.screenleft = exp[0].evalI(c)
			shouldResetCamera = true
		case modifyStageVar_bound_screenright:
			s.screenright = exp[0].evalI(c)
			shouldResetCamera = true
		// StageInfo group
		case modifyStageVar_stageinfo_zoffset:
			s.stageCamera.zoffset = exp[0].evalI(c)
			shouldResetCamera = true
		case modifyStageVar_stageinfo_zoffsetlink:
			s.zoffsetlink = exp[0].evalI(c)
		case modifyStageVar_stageinfo_xscale:
			s.scale[0] = exp[0].evalF(c)
		case modifyStageVar_stageinfo_yscale:
			s.scale[1] = exp[0].evalF(c)
		// Shadow group
		case modifyStageVar_shadow_intensity:
			s.sdw.intensity = Clamp(exp[0].evalI(c), 0, 255)
		case modifyStageVar_shadow_color:
			r := Clamp(exp[0].evalI(c), 0, 255)
			g := Clamp(exp[1].evalI(c), 0, 255)
			b := Clamp(exp[2].evalI(c), 0, 255)
			s.sdw.color = uint32(r<<16 | g<<8 | b)
		case modifyStageVar_shadow_yscale:
			s.sdw.yscale = exp[0].evalF(c)
		case modifyStageVar_shadow_fade_range:
			s.sdw.fadeend = exp[0].evalI(c)
			s.sdw.fadebgn = exp[1].evalI(c)
		case modifyStageVar_shadow_xshear:
			s.sdw.xshear = exp[0].evalF(c)
		case modifyStageVar_shadow_offset:
			s.sdw.offset[0] = exp[0].evalF(c)
			s.sdw.offset[1] = exp[1].evalF(c)
		case modifyStageVar_shadow_window:
			s.sdw.window[0] = exp[0].evalF(c)
			s.sdw.window[1] = exp[1].evalF(c)
			s.sdw.window[2] = exp[2].evalF(c)
			s.sdw.window[3] = exp[3].evalF(c)
		// Reflection group
		case modifyStageVar_reflection_intensity:
			s.reflection.intensity = Clamp(exp[0].evalI(c), 0, 255)
		case modifyStageVar_reflection_yscale:
			s.reflection.yscale = exp[0].evalF(c)
		case modifyStageVar_reflection_xshear:
			s.reflection.xshear = exp[0].evalF(c)
		case modifyStageVar_reflection_color:
			r := Clamp(exp[0].evalI(c), 0, 255)
			g := Clamp(exp[1].evalI(c), 0, 255)
			b := Clamp(exp[2].evalI(c), 0, 255)
			s.reflection.color = uint32(r<<16 | g<<8 | b)
		case modifyStageVar_reflection_offset:
			s.reflection.offset[0] = exp[0].evalF(c)
			s.reflection.offset[1] = exp[1].evalF(c)
		case modifyStageVar_reflection_window:
			s.reflection.window[0] = exp[0].evalF(c)
			s.reflection.window[1] = exp[1].evalF(c)
			s.reflection.window[2] = exp[2].evalF(c)
			s.reflection.window[3] = exp[3].evalF(c)
		}
		return true
	})
	s.reload = true // Stage will have to be reloaded if it's re-selected
	if shouldResetCamera {
		sys.cam.stageCamera = s.stageCamera
		sys.cam.Reset() // TODO: Resetting the camera makes the zoom jitter
	}
	return false
}

type cameraCtrl StateControllerBase

const (
	cameraCtrl_view byte = iota
	cameraCtrl_pos
	cameraCtrl_followid
)

func (sc cameraCtrl) Run(c *Char, _ []int32) bool {
	//crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case cameraCtrl_view:
			sys.cam.View = CameraView(exp[0].evalI(c))
			if sys.cam.View == Follow_View {
				sys.cam.FollowChar = c
			}
		case cameraCtrl_pos:
			sys.cam.Pos[0] = exp[0].evalF(c)
			if len(exp) > 1 {
				sys.cam.Pos[1] = exp[1].evalF(c)
			}
		case cameraCtrl_followid:
			if cid := sys.playerID(exp[0].evalI(c)); cid != nil {
				sys.cam.FollowChar = cid
			}
		}
		return true
	})
	return false
}

type height StateControllerBase

const (
	height_value byte = iota
	height_redirectid
)

func (sc height) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case height_value:
			var v1, v2 float32
			v1 = exp[0].evalF(c)
			if len(exp) > 1 {
				v2 = exp[1].evalF(c)
			}
			crun.setHeight(v1*redirscale, v2*redirscale)
		case height_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = (320 / c.localcoord) / (320 / crun.localcoord)
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type depth StateControllerBase

const (
	depth_edge byte = iota
	depth_player
	depth_value
	depth_redirectid
)

func (sc depth) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case depth_player:
			var v1, v2 float32
			v1 = exp[0].evalF(c)
			if len(exp) > 1 {
				v2 = exp[1].evalF(c)
			}
			crun.setDepth(v1*redirscale, v2*redirscale)
		case depth_edge:
			var v1, v2 float32
			v1 = exp[0].evalF(c)
			if len(exp) > 1 {
				v2 = exp[1].evalF(c)
			}
			crun.setDepthEdge(v1*redirscale, v2*redirscale)
		case depth_value:
			var v1, v2 float32
			v1 = exp[0].evalF(c)
			if len(exp) > 1 {
				v2 = exp[1].evalF(c)
			}
			crun.setDepth(v1*redirscale, v2*redirscale)
			crun.setDepthEdge(v1*redirscale, v2*redirscale)
		case depth_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = (320 / c.localcoord) / (320 / crun.localcoord)
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type modifyPlayer StateControllerBase

const (
	modifyPlayer_lifemax byte = iota
	modifyPlayer_powermax
	modifyPlayer_dizzypointsmax
	modifyPlayer_guardpointsmax
	modifyPlayer_teamside
	modifyPlayer_displayname
	modifyPlayer_lifebarname
	modifyPlayer_helperid
	modifyPlayer_helpername
	modifyPlayer_movehit
	modifyPlayer_moveguarded
	modifyPlayer_movereversed
	modifyPlayer_movecountered
	modifyPlayer_hitpausetime
	modifyPlayer_pausemovetime
	modifyPlayer_supermovetime
	modifyPlayer_redirectid
)

// TODO: Undo all effects if a cached character is loaded
func (sc modifyPlayer) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyPlayer_lifemax:
			lm := exp[0].evalI(c)
			if lm < 1 {
				lm = 1
			}
			crun.lifeMax = lm
			crun.life = Clamp(crun.life, 0, crun.lifeMax)
		case modifyPlayer_powermax:
			pm := exp[0].evalI(c)
			if pm < 0 {
				pm = 0
			}
			crun.powerMax = pm
			crun.power = Clamp(crun.power, 0, crun.powerMax)
		case modifyPlayer_dizzypointsmax:
			dp := exp[0].evalI(c)
			if dp < 0 {
				dp = 0
			}
			crun.dizzyPointsMax = dp
			crun.dizzyPoints = Clamp(crun.dizzyPoints, 0, crun.dizzyPointsMax)
		case modifyPlayer_guardpointsmax:
			gp := exp[0].evalI(c)
			if gp < 0 {
				gp = 0
			}
			crun.guardPointsMax = gp
			crun.guardPoints = Clamp(crun.guardPoints, 0, crun.guardPointsMax)
		case modifyPlayer_teamside:
			ts := int(exp[0].evalI(c)) - 1 // Internally the teamside starts at -1 instead of 0
			if ts >= -1 && ts <= 1 && ts != crun.teamside {
				crun.teamside = ts
				// Reevaluate alliances
				if crun.playerFlag {
					sys.charList.enemyNearChanged = true
				} else {
					crun.enemyNearP2Clear()
				}
			}
		case modifyPlayer_displayname:
			dn := string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.cgi[crun.playerNo].displayname = dn
		case modifyPlayer_lifebarname:
			ln := string(*(*[]byte)(unsafe.Pointer(&exp[0])))
			sys.cgi[crun.playerNo].lifebarname = ln
		case modifyPlayer_helperid:
			if crun.helperIndex != 0 {
				id := exp[0].evalI(c)
				if id >= 0 {
					crun.helperId = id
				} else {
					crun.helperId = 0
				}
			}
		case modifyPlayer_helpername:
			if crun.helperIndex != 0 {
				hn := string(*(*[]byte)(unsafe.Pointer(&exp[0])))
				crun.name = hn
			}
		case modifyPlayer_movehit:
			crun.mctype = MC_Hit
			crun.mctime = Max(0, exp[0].evalI(c))
		case modifyPlayer_moveguarded:
			crun.mctype = MC_Guarded
			crun.mctime = Max(0, exp[0].evalI(c))
		case modifyPlayer_movereversed:
			crun.mctype = MC_Reversed
			crun.mctime = Max(0, exp[0].evalI(c))
		case modifyPlayer_movecountered:
			crun.counterHit = exp[0].evalB(c)
		case modifyPlayer_hitpausetime:
			crun.hitPauseTime = Max(0, exp[0].evalI(c))
		case modifyPlayer_pausemovetime:
			crun.pauseMovetime = Max(0, exp[0].evalI(c))
		case modifyPlayer_supermovetime:
			crun.superMovetime = Max(0, exp[0].evalI(c))
		case modifyPlayer_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type getHitVarSet StateControllerBase

const (
	getHitVarSet_airtype byte = iota
	getHitVarSet_animtype
	getHitVarSet_attr
	getHitVarSet_chainid
	getHitVarSet_ctrltime
	getHitVarSet_down_recover
	getHitVarSet_down_recovertime
	getHitVarSet_fall
	getHitVarSet_fall_damage
	getHitVarSet_fall_envshake_ampl
	getHitVarSet_fall_envshake_freq
	getHitVarSet_fall_envshake_mul
	getHitVarSet_fall_envshake_phase
	getHitVarSet_fall_envshake_time
	getHitVarSet_fall_envshake_dir
	getHitVarSet_fall_kill
	getHitVarSet_fall_recover
	getHitVarSet_fall_recovertime
	getHitVarSet_fall_xvel
	getHitVarSet_fall_yvel
	getHitVarSet_fall_zvel
	getHitVarSet_fallcount
	getHitVarSet_ground_animtype
	getHitVarSet_groundtype
	getHitVarSet_guarded
	getHitVarSet_hitshaketime
	getHitVarSet_hittime
	getHitVarSet_id
	getHitVarSet_playerno
	getHitVarSet_recovertime
	getHitVarSet_slidetime
	getHitVarSet_xvel
	getHitVarSet_yvel
	getHitVarSet_zvel
	getHitVarSet_xaccel
	getHitVarSet_yaccel
	getHitVarSet_zaccel
	getHitVarSet_redirectid
)

func (sc getHitVarSet) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case getHitVarSet_airtype:
			crun.ghv.airtype = HitType(exp[0].evalI(c))
		case getHitVarSet_animtype:
			crun.ghv.animtype = Reaction(exp[0].evalI(c))
		case getHitVarSet_attr:
			crun.ghv.attr = exp[0].evalI(c)
		case getHitVarSet_chainid:
			crun.ghv.hitid = exp[0].evalI(c)
		case getHitVarSet_ctrltime:
			crun.ghv.ctrltime = exp[0].evalI(c)
		case getHitVarSet_down_recover:
			crun.ghv.down_recover = exp[0].evalB(c)
		case getHitVarSet_down_recovertime:
			crun.ghv.down_recovertime = exp[0].evalI(c)
		case getHitVarSet_fall:
			crun.ghv.fallflag = exp[0].evalB(c)
		case getHitVarSet_fall_damage:
			crun.ghv.fall_damage = exp[0].evalI(c)
		case getHitVarSet_fall_envshake_ampl:
			crun.ghv.fall_envshake_ampl = int32(exp[0].evalF(c) * redirscale)
		case getHitVarSet_fall_envshake_freq:
			crun.ghv.fall_envshake_freq = exp[0].evalF(c)
		case getHitVarSet_fall_envshake_mul:
			crun.ghv.fall_envshake_mul = exp[0].evalF(c)
		case getHitVarSet_fall_envshake_dir:
			crun.ghv.fall_envshake_dir = exp[0].evalF(c)
		case getHitVarSet_fall_envshake_phase:
			crun.ghv.fall_envshake_phase = exp[0].evalF(c)
		case getHitVarSet_fall_envshake_time:
			crun.ghv.fall_envshake_time = exp[0].evalI(c)
		case getHitVarSet_fall_kill:
			crun.ghv.fall_kill = exp[0].evalB(c)
		case getHitVarSet_fall_recover:
			crun.ghv.fall_recover = exp[0].evalB(c)
		case getHitVarSet_fall_recovertime:
			crun.ghv.fall_recovertime = exp[0].evalI(c)
		case getHitVarSet_fall_xvel:
			crun.ghv.fall_xvelocity = exp[0].evalF(c) * redirscale
		case getHitVarSet_fall_yvel:
			crun.ghv.fall_yvelocity = exp[0].evalF(c) * redirscale
		case getHitVarSet_fall_zvel:
			crun.ghv.fall_zvelocity = exp[0].evalF(c) * redirscale
		case getHitVarSet_fallcount:
			crun.ghv.fallcount = exp[0].evalI(c)
		case getHitVarSet_groundtype:
			crun.ghv.groundtype = HitType(exp[0].evalI(c))
		case getHitVarSet_guarded:
			crun.ghv.guarded = exp[0].evalB(c)
		case getHitVarSet_hittime:
			crun.ghv.hittime = exp[0].evalI(c)
		case getHitVarSet_hitshaketime:
			crun.ghv.hitshaketime = exp[0].evalI(c)
		case getHitVarSet_id:
			crun.ghv.playerId = exp[0].evalI(c)
		case getHitVarSet_playerno:
			crun.ghv.playerNo = int(exp[0].evalI(c))
		case getHitVarSet_slidetime:
			crun.ghv.slidetime = exp[0].evalI(c)
		case getHitVarSet_xvel:
			crun.ghv.xvel = exp[0].evalF(c) * redirscale
		case getHitVarSet_yvel:
			crun.ghv.yvel = exp[0].evalF(c) * redirscale
		case getHitVarSet_zvel:
			crun.ghv.zvel = exp[0].evalF(c) * redirscale
		case getHitVarSet_xaccel:
			crun.ghv.xaccel = exp[0].evalF(c) * redirscale
		case getHitVarSet_yaccel:
			crun.ghv.yaccel = exp[0].evalF(c) * redirscale
		case getHitVarSet_zaccel:
			crun.ghv.zaccel = exp[0].evalF(c) * redirscale
		case getHitVarSet_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type groundLevelOffset StateControllerBase

const (
	groundLevelOffset_value byte = iota
	groundLevelOffset_redirectid
)

func (sc groundLevelOffset) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case groundLevelOffset_value:
			crun.groundLevel = exp[0].evalF(c) * redirscale
		case groundLevelOffset_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type targetAdd StateControllerBase

const (
	targetAdd_playerid byte = iota
	targetAdd_redirectid
)

func (sc targetAdd) Run(c *Char, _ []int32) bool {
	crun := c
	var pid int32
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case targetAdd_playerid:
			pid = exp[0].evalI(c)
		case targetAdd_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		// Check if ID exists
		if pid > 0 {
			done := false
			for i := range sys.chars {
				for j := range sys.chars[i] {
					if sys.chars[i][j].id == pid {
						// Add target to char's "target" list
						// This function already prevents duplicating targets
						crun.addTarget(pid)
						// Add char to target's "hit by" list
						// Keep juggle points if target already exists
						jug := crun.gi().data.airjuggle
						for _, v := range sys.chars[i][j].ghv.hitBy {
							if v[0] == crun.id {
								jug = v[1]
							}
						}
						// Remove then readd char to the list with the new juggle points
						sys.chars[i][j].ghv.dropId(crun.id)
						sys.chars[i][j].ghv.hitBy = append(sys.chars[i][j].ghv.hitBy, [...]int32{crun.id, jug})
						done = true
						break
					}
				}
				if done {
					break
				}
			}
		}
		return true
	})
	return false
}

type transformClsn StateControllerBase

const (
	transformClsn_scale byte = iota
	transformClsn_angle
	transformClsn_redirectid
)

func (sc transformClsn) Run(c *Char, _ []int32) bool {
	crun := c
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case transformClsn_scale:
			crun.clsnScaleMul[0] *= exp[0].evalF(c)
			if len(exp) > 1 {
				crun.clsnScaleMul[1] *= exp[1].evalF(c)
			}
			crun.updateClsnScale()
		case transformClsn_angle:
			crun.clsnAngle += exp[0].evalF(c)
		case transformClsn_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type transformSprite StateControllerBase

const (
	transformSprite_window byte = iota
	transformSprite_xshear
	transformSprite_redirectid
)

func (sc transformSprite) Run(c *Char, _ []int32) bool {
	crun := c
	var redirscale float32 = 1.0
	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case transformSprite_window:
			crun.window = [4]float32{exp[0].evalF(c) * redirscale, exp[1].evalF(c) * redirscale, exp[2].evalF(c) * redirscale, exp[3].evalF(c) * redirscale}
		case transformSprite_xshear:
			crun.xshear = exp[0].evalF(c)
		case transformSprite_redirectid:
			if rid := sys.playerID(exp[0].evalI(c)); rid != nil {
				crun = rid
				redirscale = c.localscl / crun.localscl
			} else {
				return false
			}
		}
		return true
	})
	return false
}

type modifyStageBG StateControllerBase

const (
	modifyStageBG_id byte = iota
	modifyStageBG_index
	modifyStageBG_actionno
	modifyStageBG_alpha
	modifyStageBG_angle
	modifyStageBG_delta_x
	modifyStageBG_delta_y
	modifyStageBG_layerno
	modifyStageBG_pos_x
	modifyStageBG_pos_y
	modifyStageBG_spriteno
	modifyStageBG_start_x
	modifyStageBG_start_y
	modifyStageBG_scalestart
	modifyStageBG_trans
	modifyStageBG_velocity_x
	modifyStageBG_velocity_y
	modifyStageBG_xshear
)

func (sc modifyStageBG) Run(c *Char, _ []int32) bool {
	bgid := int32(-1)
	bgidx := int(-1)
	var backgrounds []*backGround

	// Helper function to modify each BG
	eachBg := func(f func(bg *backGround)) {
		for _, bg := range backgrounds {
			f(bg)
		}
	}

	StateControllerBase(sc).run(c, func(paramID byte, exp []BytecodeExp) bool {
		switch paramID {
		case modifyStageBG_id:
			bgid = exp[0].evalI(c)
		case modifyStageBG_index:
			bgidx = int(exp[0].evalI(c))
		default:
			// Get BG's to modify
			if len(backgrounds) == 0 {
				backgrounds = c.getMultipleStageBg(bgid, bgidx, false)
				if len(backgrounds) == 0 {
					return false
				}
			}
			// Start modifying
			switch paramID {
			case modifyStageBG_actionno:
				val := exp[0].evalI(c)
				a := sys.stage.at.get(val) // Check if stage has that animation
				if a != nil {
					eachBg(func(bg *backGround) {
						if bg._type == BG_Anim {
							bg.changeAnim(val, a)
							bg.anim.Action() // This step is necessary because stages update before characters
						}
					})
				}
			case modifyStageBG_alpha:
				v1 := int16(exp[0].evalI(c))
				v2 := int16(exp[1].evalI(c))
				eachBg(func(bg *backGround) {
					bg.anim.mask = 0
					bg.anim.srcAlpha = v1
					bg.anim.dstAlpha = v2
					if bg.anim.srcAlpha == 1 && bg.anim.dstAlpha == 255 { // Sub fix
						bg.anim.srcAlpha = 0
					}
				})
			case modifyStageBG_angle:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.angle = val
				})
			case modifyStageBG_delta_x:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.delta[0] = val
				})
			case modifyStageBG_delta_y:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.delta[1] = val
				})
			case modifyStageBG_layerno:
				val := exp[0].evalI(c)
				eachBg(func(bg *backGround) {
					bg.layerno = val
				})
			case modifyStageBG_pos_x:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.bga.pos[0] = val
				})
			case modifyStageBG_pos_y:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.bga.pos[1] = val
				})
			case modifyStageBG_spriteno:
				gr := exp[0].evalI(c)
				im := exp[1].evalI(c)
				eachBg(func(bg *backGround) {
					if bg._type == BG_Normal {
						bg.anim.frames = []AnimFrame{*newAnimFrame()}
						bg.anim.frames[0].Group = I32ToI16(gr)
						bg.anim.frames[0].Number = I32ToI16(im)
					}
				})
			case modifyStageBG_start_x:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.start[0] = val
				})
			case modifyStageBG_start_y:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.start[1] = val
				})
			case modifyStageBG_scalestart:
				sclx := exp[0].evalF(c)
				scly := exp[1].evalF(c)
				eachBg(func(bg *backGround) {
					bg.scalestart[0] = sclx
					bg.scalestart[1] = scly
				})
			case modifyStageBG_trans:
				val := exp[0].evalI(c)
				if val == 0 || val == 1 || val == 2 || val == 3 || val == 4 {
					eachBg(func(bg *backGround) {
						switch val {
						case 0: // None
							bg.anim.srcAlpha = -1
							bg.anim.dstAlpha = 0
						case 1: // Add
							bg.anim.mask = 0
							bg.anim.srcAlpha = 255
							bg.anim.dstAlpha = 255
						case 2: // Add1
							bg.anim.mask = 0
							bg.anim.srcAlpha = 255
							bg.anim.dstAlpha = 128
						case 3: // Addalpha
							bg.anim.mask = 0
							bg.anim.srcAlpha = 255 // Default to Add first
							bg.anim.dstAlpha = 255
						case 4: // Sub
							bg.anim.mask = 0
							bg.anim.srcAlpha = 1 // That old hack that needs refactoring
							bg.anim.dstAlpha = 255
						}
					})
				}
			case modifyStageBG_velocity_x:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.bga.vel[0] = val
				})
			case modifyStageBG_velocity_y:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.bga.vel[1] = val
				})
			case modifyStageBG_xshear:
				val := exp[0].evalF(c)
				eachBg(func(bg *backGround) {
					bg.xshear = val
				})
			}
		}
		return true
	})
	return false
}

// StateDef data struct
type StateBytecode struct {
	stateType StateType
	moveType  MoveType
	physics   StateType
	playerNo  int
	stateDef  stateDef
	block     StateBlock
	ctrlsps   []int32
	numVars   int32
}

// StateDef bytecode creation function
func newStateBytecode(pn int) *StateBytecode {
	sb := &StateBytecode{
		stateType: ST_S,
		moveType:  MT_I,
		physics:   ST_N,
		playerNo:  pn,
		block:     *newStateBlock(),
	}
	return sb
}

func (sb *StateBytecode) init(c *Char) {
	if sb.stateType != ST_U {
		c.ss.changeStateType(sb.stateType)
	}
	if sb.moveType != MT_U {
		if !c.ss.storeMoveType {
			c.ss.prevMoveType = c.ss.moveType
		}
		c.ss.moveType = sb.moveType
	}
	if sb.physics != ST_U {
		c.ss.physics = sb.physics
	}
	c.ss.storeMoveType = false
	sys.workingState = sb
	sb.stateDef.Run(c)
}

func (sb *StateBytecode) run(c *Char) (changeState bool) {
	sys.bcVar = sys.bcVarStack.Alloc(int(sb.numVars))
	sys.workingState = sb
	changeState = sb.block.Run(c, sb.ctrlsps)
	if len(sys.bcStack) != 0 {
		sys.errLog.Println(sys.cgi[sb.playerNo].def)
		for _, v := range sys.bcStack {
			sys.errLog.Printf("%+v\n", v)
		}
		c.panic()
	}
	sys.bcVarStack.Clear()
	return
}
