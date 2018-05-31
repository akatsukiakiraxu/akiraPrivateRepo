/*******************************************************************************
Copyright (c) 2018 Fixstars Co., Ltd.
********************************************************************************

================================================================================
REVISION HISTORY


================================================================================
--------------------------------------------------------------------------------
$Log:  $
--------------------------------------------------------------------------------
$NoKeywords:  $
*/
/******************************************************************************
*
* File name:
*    ChangeSettingUtility.hpp
*
* Description:
*
******************************************************************************/
#ifndef _CHANGESETTING_UTILITY_HPP_
#define _CHANGESETTING_UTILITY_HPP_
#include <string>
#include <stdint.h>
#include <sys/stat.h>
#include <map>

using namespace std;
/*
type TriggerSettings struct {
	ChannelName string      `json:"channel_name"`
	Threshold   float64     `json:"threshold"`
	Hysteresis  float64     `json:"hysteresis"`
	DeadTime    float64     `json:"dead_time"`
	Timeout     float64     `json:"timeout"`
	Mode        TriggerMode `json:"trigger_mode"`
}
*/
typedef struct triggerSettings {
	int channelIndex;
	double threshold;
	double hysteresis;
	double deadTime;
	double timeout;
	int mode;
} triggerSettings_t;

typedef struct channel {
    bool enabled;
    float sampling_rate;
    string range;
    string coupling;
    int signal_input_type;
} channel_t;

typedef struct setting {
    bool enabled;
    bool comparing;
    triggerSettings_t trigger;
    channel_t channel[16];
} setting_t;

class ChangeSettingUtility
{
public:
    ChangeSettingUtility();
    virtual ~ChangeSettingUtility();

private:
    // variable
    int m_channleCount;
    string m_settingFileName;
    setting_t m_setting;
    map<string, string> m_valueRangeTbl;
    map<string, int> m_signalInputTypeTbl;
    map<string, int> m_channleNameTbl;
    map<string, int> m_triggerModeTbl;

public:
    void setSettingFileName(string settingFileName);
    void loadSetting();
    bool isExistSettingFile();
    inline setting_t & getSetting()
    {
        return (m_setting);
    }
    void dumpSetting();

    /* this function is just for debug */
    bool isHoldSettingFile();
};

#endif // _CHANGESETTING_UTILITY_HPP_
