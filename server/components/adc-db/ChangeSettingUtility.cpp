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
*    ChangeSettingUtility.cpp
*
* Description:
*
******************************************************************************/
#include <iostream>
#include <string>
#include <boost/format.hpp>
#include <boost/foreach.hpp>
#include <boost/property_tree/ptree.hpp>
#include <boost/property_tree/json_parser.hpp>

#include "ChangeSettingUtility.hpp"
#include "PropertyUtility.hpp"
#include "AsyncCommand.hpp"
#include "debug.h"

/*
 * UTF-8でないとmapのKeyが文字化けするため、ファイルのエンコードはUTF-8にすること
 */
ChangeSettingUtility::ChangeSettingUtility() {
    PropertyUtility& propertyUtility = Singleton<PropertyUtility>::GetInstance();
    m_channleCount = propertyUtility.getDeviceInfomation().SampleChannelCount;

    m_valueRangeTbl.insert(make_pair(u8"±10[V]", u8"+-10V"));
    m_valueRangeTbl.insert(make_pair(u8"±5[V]", u8"+-5V"));
    m_valueRangeTbl.insert(make_pair(u8"±2.5[V]", u8"+-2.5V"));
    m_valueRangeTbl.insert(make_pair(u8"±1.25[V]", u8"+-1.25V"));
    m_valueRangeTbl.insert(make_pair(u8"±0.625[V]", u8"+-625mV"));
    m_valueRangeTbl.insert(make_pair(u8"0~+10[V]", u8"0-10V"));
    m_valueRangeTbl.insert(make_pair(u8"0~+5[V]", u8"0-5V"));
    m_valueRangeTbl.insert(make_pair(u8"0~+2.5[V]", u8"0-2.5V"));
    m_valueRangeTbl.insert(make_pair(u8"0~+1.25[V]", u8"0-1.25V"));

    m_signalInputTypeTbl.insert(make_pair(u8"single_ended", 0));
    m_signalInputTypeTbl.insert(make_pair(u8"differential", 1));

    for (int i = 0; i < m_channleCount; ++i) {
        std::string chname = (boost::format("%1%%2%") % std::string("ch") % (i+1)).str();
        m_channleNameTbl.insert(make_pair(chname, i));
    }

    m_triggerModeTbl.insert(make_pair(u8"disabled", EDGE_NOTFOUND));
    m_triggerModeTbl.insert(make_pair(u8"falling", EDGE_FALLINGDOWN));
    m_triggerModeTbl.insert(make_pair(u8"rising", EDGE_RASINGUP));
    m_triggerModeTbl.insert(make_pair(u8"both", EDGE_BOTH));

}

ChangeSettingUtility::~ChangeSettingUtility() {
}

void ChangeSettingUtility::setSettingFileName(string settingFileName) {
    m_settingFileName = settingFileName;
}

bool ChangeSettingUtility::isExistSettingFile() {
    struct stat buffer;
    return (stat(m_settingFileName.c_str(), &buffer) == 0);
}

/* this function is just for debug */
bool ChangeSettingUtility::isHoldSettingFile() {
    struct stat buffer;
    return (stat("/tmp/olive_hold_setting", &buffer) == 0);
}

void ChangeSettingUtility::loadSetting() {
    boost::property_tree::ptree t;
    read_json(m_settingFileName, t);

    m_setting.enabled = t.get<bool>("enabled");
    m_setting.comparing = t.get<bool>("comparing");
    m_setting.trigger.channelIndex = m_channleNameTbl[t.get<string>("trigger.channel_name")];
    m_setting.trigger.threshold = t.get<double>("trigger.threshold");
    m_setting.trigger.hysteresis = t.get<double>("trigger.hysteresis");
    m_setting.trigger.deadTime = t.get<double>("trigger.dead_time");
    m_setting.trigger.timeout = t.get<double>("trigger.timeout");
    m_setting.trigger.mode = m_triggerModeTbl[t.get<string>("trigger.trigger_mode")];
    for (int i = 0; i < m_channleCount; ++i) {
        std::string chname = (boost::format("%1%%2%") % std::string("ch") % (i+1)).str();
        std::string key = "channels." + chname;
        m_setting.channel[i].enabled = t.get<bool>(key + ".enabled");
        m_setting.channel[i].sampling_rate = t.get<float>(key + ".sampling_rate");
        std::string range = t.get<string>(key + ".range");
        if (m_valueRangeTbl.count(range) > 0) {
            m_setting.channel[i].range = m_valueRangeTbl[range];
        } else {
            ADCDB_ERR("bad value range %s!", range.c_str());
        }
        m_setting.channel[i].coupling = t.get<string>(key + ".coupling");

        std::string signalInputType = t.get<string>(key + ".signal_input_type");
        if (m_signalInputTypeTbl.count(signalInputType) > 0) {
            m_setting.channel[i].signal_input_type = m_signalInputTypeTbl[signalInputType];
        }else {
            ADCDB_ERR("bad value signal_input_type %s!", signalInputType.c_str());
        }
    }
}

void ChangeSettingUtility::dumpSetting() {
    ADCDB_INFO("m_setting.enabled=%d", m_setting.enabled);
    ADCDB_INFO("m_setting.comparing=%d", m_setting.comparing);
    ADCDB_INFO("m_setting.trigger.channelIndex=%d", m_setting.trigger.channelIndex);
    ADCDB_INFO("m_setting.trigger.threshold=%f", m_setting.trigger.threshold);
    ADCDB_INFO("m_setting.trigger.hysteresis=%f", m_setting.trigger.hysteresis);
    ADCDB_INFO("m_setting.trigger.deadTime=%f", m_setting.trigger.deadTime);
    ADCDB_INFO("m_setting.trigger.timeout=%f", m_setting.trigger.timeout);
    ADCDB_INFO("m_setting.trigger.mode=%d", m_setting.trigger.mode);

    for (int i = 0; i < m_channleCount; ++i) {
        ADCDB_INFO("m_setting.channel[%d].enabled=%d", i+1 ,m_setting.channel[i].enabled);
        ADCDB_INFO("m_setting.channel[%d].sampling_rate=%f", i+1 ,m_setting.channel[i].sampling_rate);
        ADCDB_INFO("m_setting.channel[%d].range=%s", i+1 ,m_setting.channel[i].range.c_str());
        ADCDB_INFO("m_setting.channel[%d].coupling=%s", i+1 ,m_setting.channel[i].coupling.c_str());
        ADCDB_INFO("m_setting.channel[%d].signal_input_type=%d", i+1 ,m_setting.channel[i].signal_input_type);
   }
}
