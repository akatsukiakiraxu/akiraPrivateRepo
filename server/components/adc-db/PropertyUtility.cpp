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
*    PropertyUtility.cpp
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

#include "PropertyUtility.hpp"
PropertyUtility::PropertyUtility() {
}

PropertyUtility::~PropertyUtility() {
}

void PropertyUtility::setPropertyFileName(char* propertyFileName) {
    m_propertyFileName = string(propertyFileName);
}

bool PropertyUtility::isExistPropertyFile() {
    struct stat buffer;
    return (stat(m_propertyFileName.c_str(), &buffer) == 0);
}

void PropertyUtility::getProperties() {
    boost::property_tree::ptree t;
    read_json(m_propertyFileName, t);

    string str_desc = t.get<string>("Device.Description");
    wchar_t* wcs = new wchar_t[str_desc.length() + 1];
    mbstowcs(wcs, str_desc.c_str(), str_desc.length() + 1);
    m_device.Description = wstring(wcs);
    delete [] wcs;

    m_valueRangeMap.insert(make_pair(u8"+-10V", 1));
    m_valueRangeMap.insert(make_pair(u8"+-5V", 2));
    m_valueRangeMap.insert(make_pair(u8"+-2.5V", 3));
    m_valueRangeMap.insert(make_pair(u8"+-1.25V", 4));
    m_valueRangeMap.insert(make_pair(u8"0-10V", 7));
    m_valueRangeMap.insert(make_pair(u8"0-5V", 8));
    m_valueRangeMap.insert(make_pair(u8"0-2.5V", 9));
    m_valueRangeMap.insert(make_pair(u8"0-1.25V", 10));
    m_valueRangeMap.insert(make_pair(u8"+-625mV", 12));

    /*
    * Device
    */
    m_device.StartChannel = t.get<int>("Device.StartChannel");
    m_device.SampleChannelCount = t.get<int>("Device.SampleChannelCount");
    m_device.ChannelCountMax = t.get<int>("Device.ChannelCountMax");
    m_device.IntervalCount = t.get<int>("Device.IntervalCount");
    m_device.ValueRange = m_valueRangeMap[t.get<string>("Device.ValueRange")];
    m_device.SignalType = t.get<int>("Device.SignalType");
    m_device.SamplingRate = t.get<double>("Device.SamplingRate");

    /*
    * Soft
    */
    m_softCtrl.IsOutputSummaryData = t.get<bool>("SoftCtrl.IsOutputSummaryData");
    m_softCtrl.IsOutputRawData = t.get<bool>("SoftCtrl.IsOutputRawData");
    m_softCtrl.SoftTriggerType = t.get<int>("SoftCtrl.SoftTriggerType");
    m_softCtrl.SoftTrigger.IntervalMs = t.get<int>("SoftCtrl.SoftTrigger.IntervalMs");
    m_softCtrl.SoftTrigger.Threshold = t.get<double>("SoftCtrl.SoftTrigger.Threshold");
    m_softCtrl.SoftTrigger.ThresholdDeviation = t.get<double>("SoftCtrl.SoftTrigger.ThresholdDeviation");
    m_softCtrl.SoftTrigger.ThresholdMaskTime = t.get<float>("SoftCtrl.SoftTrigger.ThresholdMaskTime");
    m_softCtrl.SoftTrigger.ChannelIndex = t.get<int>("SoftCtrl.SoftTrigger.ChannelIndex");
    m_softCtrl.SoftTrigger.EdgeType = t.get<int>("SoftCtrl.SoftTrigger.EdgeType");
    m_softCtrl.RecordTimeout = t.get<float>("SoftCtrl.RecordTimeout");
    m_softCtrl.IsDumpData = t.get<bool>("SoftCtrl.IsDumpData");
    m_softCtrl.DumpDataPath = t.get<string>("SoftCtrl.DumpDataPath");
    m_softCtrl.IsShowDebugLog = t.get<bool>("SoftCtrl.IsShowDebugLog");
    m_tiggerMaskPoints = static_cast<uint64_t>(m_device.SamplingRate * m_softCtrl.SoftTrigger.ThresholdMaskTime);
    m_recordTimeoutPoints = static_cast<uint64_t>(m_device.SamplingRate * m_softCtrl.RecordTimeout);

    /*
    * Data Format
    */
    m_adcDataFmt.SummaryData.DataCountPerChannel = t.get<int>("ADCDataFmt.SummaryData.DataCountPerChannel");
    m_adcDataFmt.SummaryData.Type = t.get<int>("ADCDataFmt.SummaryData.Type");
    m_adcDataFmt.SummaryData.DataSize = t.get<int>("ADCDataFmt.SummaryData.DataSize");
    m_adcDataFmt.SummaryData.ExpectMax = t.get<double>("ADCDataFmt.SummaryData.ExpectMax");
    m_adcDataFmt.SummaryData.ExpectMin = t.get<double>("ADCDataFmt.SummaryData.ExpectMin");
    m_adcDataFmt.FFTData.Type = t.get<int>("ADCDataFmt.FFTData.Type");
    m_adcDataFmt.FFTData.ExpectMax = t.get<double>("ADCDataFmt.FFTData.ExpectMax");
    m_adcDataFmt.FFTData.ExpectMin = t.get<double>("ADCDataFmt.FFTData.ExpectMin");
    m_adcDataFmt.RawData.Type = t.get<int>("ADCDataFmt.RawData.Type");
}

void PropertyUtility::notifyChangedSetting(ChangeSettingUtility *p) {
    int channelIndex = p->getSetting().trigger.channelIndex;
    /* device property */
    m_device.SamplingRate = static_cast<double>(p->getSetting().channel[channelIndex].sampling_rate);
    m_device.ValueRange = m_valueRangeMap[p->getSetting().channel[channelIndex].range];
    m_device.SignalType = p->getSetting().channel[channelIndex].signal_input_type;

    /* soft property */
    m_softCtrl.SoftTrigger.ChannelIndex = channelIndex;
    m_softCtrl.SoftTrigger.Threshold = p->getSetting().trigger.threshold;
    m_softCtrl.SoftTrigger.EdgeType = p->getSetting().trigger.mode;
    m_softCtrl.SoftTrigger.ThresholdMaskTime = p->getSetting().trigger.deadTime;
    //m_softCtrl.SoftTrigger.ThresholdDeviation = p->getSetting().trigger.hysteresis;
    m_softCtrl.RecordTimeout = p->getSetting().trigger.timeout;
}

void PropertyUtility::saveProperties() {
    /* read json */
    boost::property_tree::ptree t;
    read_json(m_propertyFileName, t);

    /* device property */
    t.put("Device.SamplingRate", boost::format("%6.4f") % m_device.SamplingRate);
    string valueRangeStr("");
    for (auto& p : m_valueRangeMap) {
        if (p.second == m_device.ValueRange) {
            valueRangeStr = p.first;
            break;
        }
    }
    t.put("Device.ValueRange", valueRangeStr);
    t.put("Device.SignalType", m_device.SignalType);

    /* soft property */
    t.put("SoftCtrl.SoftTrigger.ChannelIndex", m_softCtrl.SoftTrigger.ChannelIndex);
    t.put("SoftCtrl.SoftTrigger.Threshold", boost::format("%6.4f") % m_softCtrl.SoftTrigger.Threshold);
    t.put("SoftCtrl.SoftTrigger.EdgeType", m_softCtrl.SoftTrigger.EdgeType);
    t.put("SoftCtrl.SoftTrigger.ThresholdMaskTime", boost::format("%6.4f") % m_softCtrl.SoftTrigger.ThresholdMaskTime);
    //t.put("SoftCtrl.SoftTrigger.ThresholdDeviation", boost::format("%6.4f") % m_softCtrl.SoftTrigger.ThresholdDeviation);
    t.put("SoftCtrl.RecordTimeout", m_softCtrl.SoftTrigger.ChannelIndex);

    /* write json */
    write_json(m_propertyFileName, t);
}
