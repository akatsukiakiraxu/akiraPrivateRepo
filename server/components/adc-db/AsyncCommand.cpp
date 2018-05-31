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
*    AsyncCommad.cpp
*
* Description:
*
******************************************************************************/
#include "AsyncCommand.hpp"
#include <stdlib.h>
#include <vector>
#include <time.h>
#include <algorithm>
#include <iterator>     // std::back_inserter
#include <map>
#include <list>

#include "bdaqctrl.h"
#include "debug.h"
#include "Singleton.hpp"

using namespace std;

enum {
    m_high = 0,
    m_middle = 1,
    m_low = 2,
};

const struct {
    int preStatus;
    int pointStatus;
    int nowStatus;
    int edgeType;
} m_status_table[15] = {
    {TS_HIGH, m_high, TS_HIGH, EDGE_NOTFOUND},
    {TS_HIGH, m_middle, TS_WAIT_LOW, EDGE_NOTFOUND},
    {TS_HIGH, m_low, TS_LOW, EDGE_FALLINGDOWN},

    {TS_LOW, m_high, TS_HIGH, EDGE_RASINGUP},
    {TS_LOW, m_middle, TS_WAIT_HIGH, EDGE_NOTFOUND},
    {TS_LOW, m_low, TS_LOW, EDGE_NOTFOUND},

    {TS_WAIT_HIGH, m_high, TS_HIGH, EDGE_RASINGUP},
    {TS_WAIT_HIGH, m_middle, TS_WAIT_HIGH, EDGE_NOTFOUND},
    {TS_WAIT_HIGH, m_low, TS_WAIT_HIGH, EDGE_NOTFOUND},

    {TS_WAIT_LOW, m_high, TS_WAIT_LOW, EDGE_NOTFOUND},
    {TS_WAIT_LOW, m_middle, TS_WAIT_LOW, EDGE_NOTFOUND},
    {TS_WAIT_LOW, m_low, TS_LOW, EDGE_FALLINGDOWN},

    {TS_UNKNOWN, m_high, TS_HIGH, EDGE_NOTFOUND},
    {TS_UNKNOWN, m_middle, TS_UNKNOWN, EDGE_NOTFOUND},
    {TS_UNKNOWN, m_low, TS_LOW, EDGE_NOTFOUND},
};


AsyncCommand::AsyncCommand()
    : m_propertyUtility(Singleton<PropertyUtility>::GetInstance())
    , m_preValue(std::numeric_limits<double>::max())
    , m_trigger(0)
    , m_triggerStatus(TS_UNKNOWN)
    , m_sendedPoints(0)
    , m_pointsAfterLastTrigger(0) {
    m_isShowDebugMsg = m_propertyUtility.getSoftCtrl().IsShowDebugLog;
    m_tiggerMaskPoints = m_propertyUtility.getTiggerMaskPoints();
    m_recordTimeoutPoints = m_propertyUtility.getRecordTimeoutPoints();
}

AsyncCommand::~AsyncCommand() {
}

int AsyncCommand::execute() {
    ADCDB_DEBUG(m_isShowDebugMsg, "cmd start");
    if (m_dataInfo.pDataBuf != NULL) {
        uint32_t dataCount = m_dataInfo.dataSize / sizeof(double);
        uint32_t dataSize = m_dataInfo.dataSize;
        uint32_t chanCount = m_dataInfo.chanCount;
        uint32_t countPerChannel = dataCount / chanCount;
        vector<vector<double> > vec_scaledData(chanCount);
        vec_scaledData.clear();
        vector<double> vec_summaryDatas(chanCount * countPerChannel);
        vec_summaryDatas.clear();
        uint16_t type = 0;
        uint32_t summaryDataSize = 0, rawDataSize = 0;
        uint64_t channelMap =  0;
        uint32_t chanStart = m_dataInfo.chanStart;
        /*********************************************
        ******* Scale userDataBuffer[] to [][]***********
        * For example
         userDataBuffer[data0, data1, data2, data3, ... data39]

         if the StartChannel = 0, ChannelCount = 4
         it will be scaled to
         CH  | DATA
         0   | data0, data4, data8, ... data36
         1   | data1, data5, data9, ... data37
         2   | data2, data6, data10, ... data38
         3   | data3, data7, data11, ... data39

         if the StartChannel = 1, ChannelCount = 2
         it will be
         CH  | DATA
         0   | data1, data3, data5, ... data39
         1   | data0, data2, data4, ... data38

        *********************************************/
        ADCDB_DEBUG(m_isShowDebugMsg, "Scale Buffer");
        for (int i = 0; i < chanCount; ++i) {
            vector<double> chanDatas(countPerChannel);
            chanDatas.clear();
            int startPos = (i >= chanStart) ? (i - chanStart) : (i + (chanCount - chanStart));
            for (int j = 0; j < countPerChannel; ++j) {
                chanDatas.push_back(m_dataInfo.pDataBuf[startPos + (j * chanCount)]);
            }
            vec_scaledData.push_back(chanDatas);
        }

        /*********************************************
        ******* SUMMARY DATA ***********
        *********************************************/
        if (m_propertyUtility.getSoftCtrl().IsOutputSummaryData) {
            ADCDB_DEBUG(m_isShowDebugMsg, "OutputSummaryData");
            type = m_propertyUtility.getAdcDataFmt().SummaryData.Type;
            summaryDataSize = chanCount * m_propertyUtility.getAdcDataFmt().SummaryData.DataCountPerChannel * sizeof(double); // 4channel 4data(max,min,expMax, expMin)
            for (vector<vector<double> >::iterator dter = vec_scaledData.begin(); dter != vec_scaledData.end(); ++dter) {
                double max = *std::max_element((*dter).begin(), (*dter).end());
                double min = *std::min_element((*dter).begin(), (*dter).end());
                vec_summaryDatas.push_back(max);
                vec_summaryDatas.push_back(min);
                vec_summaryDatas.push_back(m_propertyUtility.getAdcDataFmt().SummaryData.ExpectMax);
                vec_summaryDatas.push_back(m_propertyUtility.getAdcDataFmt().SummaryData.ExpectMin);
            }

            pipeOut(&type, sizeof(uint16_t), 1); // write type(2Bytes) to stdout
            pipeOut(&m_trigger, sizeof(uint16_t), 1); // write tiggerFlag(2Bytes) to stdout
            pipeOut(&summaryDataSize, sizeof(uint32_t), 1); // write size(4Bytes) to stdout
            for (vector<double>::iterator it = vec_summaryDatas.begin(); it != vec_summaryDatas.end(); ++it) {
                double sumValue = *it;
                pipeOut(&sumValue, sizeof(double), 1); // write summary data(1value(8Bytes)/loop) to stdout
            }
        }

        /***************************
        ******* RAW DATA ***********
        ***************************/
        double rawData[dataCount] = {0.0};
        type = m_propertyUtility.getAdcDataFmt().RawData.Type;
        uint32_t rawDataIndex = 0, channelIndex = 0;
        uint16_t trigger = TRIGGER_FRAME_MID;

        for (int i = 0; i < chanCount; ++i) {
            channelMap |= (1 << (chanStart+i));
        }
        ADCDB_DEBUG(m_isShowDebugMsg, "channelMap 0x%08X", (unsigned int)channelMap);
        rawDataSize = dataCount * sizeof(double) + sizeof(channelMap); // the sizeof rawdata is include channelMap(8Bytes)
        if (m_propertyUtility.getSoftCtrl().IsOutputRawData) {
            ADCDB_DEBUG(m_isShowDebugMsg, "OutputRawData chan[%zu] data[%zu]", vec_scaledData.size(), vec_scaledData[0].size());
            for (vector<vector<double> >::iterator dter = vec_scaledData.begin(); dter != vec_scaledData.end(); ++dter) {
                vector<double> vec_tmp = *dter;
                for (vector<double>::iterator it = vec_tmp.begin(); it != vec_tmp.end(); ++it) {
                    double value = *it;
                    rawData[rawDataIndex++] = value;
                }
                channelIndex++;
            }
        } else {
            ADCDB_DEBUG(m_isShowDebugMsg, "OutputDummyRawData 0");
            double dummyRawData[dataCount * sizeof(double)] = {0};
            pipeOut((void*)&dummyRawData[0], sizeof(double), dataCount);
        }

        vector<pair<uint16_t, uint32_t> > triggerPointVec;
        if (m_propertyUtility.getSoftCtrl().SoftTriggerType == SOFT_TRIGGER_TYPE_THRESHOLD) {
            ADCDB_DEBUG(m_isShowDebugMsg, "use soft trigger");
            vector<pair<uint16_t, uint32_t> > thresholdPointVec;
            double dev = m_propertyUtility.getSoftCtrl().SoftTrigger.ThresholdDeviation;
            double threshold = m_propertyUtility.getSoftCtrl().SoftTrigger.Threshold;
            uint32_t chIndex = m_propertyUtility.getSoftCtrl().SoftTrigger.ChannelIndex;
            int edgeType = m_propertyUtility.getSoftCtrl().SoftTrigger.EdgeType;
            thresholdPointVec = getThresholdPoint(chIndex, vec_scaledData, edgeType, threshold, dev);
            if (m_tiggerMaskPoints == 0) {
                triggerPointVec = thresholdPointVec;
            } else {
                ADCDB_DEBUG(m_isShowDebugMsg, "getTriggerMaskPoint");
                triggerPointVec = getTriggerMaskPoint(countPerChannel, thresholdPointVec);
            }
            if (triggerPointVec.size() > 0) {
                ADCDB_DEBUG(m_isShowDebugMsg, "triggerPoint[%zu]", triggerPointVec.size());
                if (m_recordTimeoutPoints > 0) {
                    m_pointsAfterLastTrigger = countPerChannel - thresholdPointVec.back().second;
                }
            } else {
                ADCDB_DEBUG(m_isShowDebugMsg, "not found trigger");
                if (m_recordTimeoutPoints > 0) {
                    m_pointsAfterLastTrigger += countPerChannel;
                    if (m_pointsAfterLastTrigger >= m_recordTimeoutPoints) {
                        triggerPointVec.push_back(make_pair(TRIGGER_FRAME_END, 1));
                        m_pointsAfterLastTrigger = 0;
                    } else {
                        /* not Timeout */
                    }
                }
            }
        } else {
            trigger = m_trigger;
        }

        if (triggerPointVec.size() > 0) {
            list<pair<uint16_t, vector<vector<double> > > > splitData;
            vector<vector<double> > sectionVec;
            uint32_t prePos = 0;
            vector<double> vec;
            int j = 0;
            trigger = TRIGGER_FRAME_MID;
            for (; j < triggerPointVec.size(); ++j) {
                uint32_t point = triggerPointVec.at(j).second;
                sectionVec.clear();
                for (int i = 0; i < chanCount; ++i) {
                    vec.clear();
                    copy(vec_scaledData.at(i).begin()+prePos, vec_scaledData.at(i).begin()+point, back_inserter(vec));
                    sectionVec.push_back(vec);
                }
                splitData.push_back(make_pair(trigger, sectionVec));
                trigger = triggerPointVec.at(j).first;
                prePos = point;
            }
            if (prePos < vec_scaledData.at(0).size()) {
                sectionVec.clear();
                for (int i = 0; i < chanCount; ++i) {
                    vec.clear();
                    copy(vec_scaledData.at(i).begin()+prePos, vec_scaledData.at(i).end(), back_inserter(vec));
                    sectionVec.push_back(vec);
                }
                splitData.push_back(make_pair(trigger, sectionVec));
            }

            for (auto& v : splitData) {
                double value = 0.0;
                int index = 0;
                uint32_t resize = 0;
                uint32_t cnt = 0;
                for (vector<vector<double> >::iterator dter = v.second.begin(); dter != v.second.end(); ++dter) {
                    vector<double> vec_tmp = *dter;
                    cnt += vec_tmp.size();
                    for (vector<double>::iterator it = vec_tmp.begin(); it != vec_tmp.end(); ++it) {
                        value = *it;
                        rawData[index++] = value;
                    }
                }
                trigger = v.first;
                if (cnt > 0) {
                    resize = cnt * sizeof(double) + sizeof(channelMap);
                    pipeOut(&type, sizeof(uint16_t), 1); // write type(2Bytes) to stdout
                    pipeOut(&trigger, sizeof(uint16_t), 1); // write tiggerFlag(2Bytes) to stdout
                    pipeOut(&resize, sizeof(uint32_t), 1);  // write size(4Bytes) to stdout
                    pipeOut(&channelMap, sizeof(uint64_t), 1); // write channelMap(8Bytes) to stdout
                    pipeOut((void*)&rawData[0], sizeof(double), cnt);
                }
            }
        } else
            if (dataCount > 0) {
                pipeOut(&type, sizeof(uint16_t), 1); // write type(2Bytes) to stdout
                pipeOut(&trigger, sizeof(uint16_t), 1); // write tiggerFlag(2Bytes) to stdout
                pipeOut(&rawDataSize, sizeof(uint32_t), 1);  // write size(4Bytes) to stdout
                pipeOut(&channelMap, sizeof(uint64_t), 1); // write channelMap(8Bytes) to stdout
                pipeOut((void*)&rawData[0], sizeof(double), dataCount); // write raw data
            } else {
                /* do nothing */
            }
    } /* if (m_dataInfo.pDataBuf != NULL)  */
    ADCDB_DEBUG(m_isShowDebugMsg, "cmd end");
    return 0;
}

void AsyncCommand::pipeOut(void* buf, size_t size, uint32_t count) {
    if (NULL != buf) {
        if (!m_isShowDebugMsg) {
            fwrite(buf, size, count, stdout);
        }
    }
}

vector<pair<uint16_t, uint32_t> > AsyncCommand::getThresholdPoint(int channelIndex, const vector<vector<double> >& data, int edgeType, double threshold, double deviation) {
    vector<pair<uint16_t, uint32_t> > frameStartPositions;
    vector<double> channelData = data.at(channelIndex);
    int pointSts = m_high, stsNow = TS_UNKNOWN, stsPre = m_triggerStatus;
    double dev = (deviation > 0) ? deviation : deviation*(-1);
    double thre1 = threshold + dev;
    double thre2 = threshold - dev;
    ADCDB_DEBUG(m_isShowDebugMsg, "thre1[%4.4f], thre2[%4.4f]", thre1, thre2);
    int point = 0;
    int edTp = EDGE_NOTFOUND;
    size_t chSize = channelData.size();

    frameStartPositions.clear();

    for (size_t index = 0; index < chSize; ++index) {
        edTp = EDGE_NOTFOUND;
        double d = channelData.at(index);
        if (d > thre1) {
            pointSts = m_high;
        } else if (d < thre2) {
            pointSts = m_low;
        } else {
            pointSts = m_middle;
        }
        ADCDB_DEBUG(m_isShowDebugMsg, "pointSts[%d]", pointSts);
        for(int i = 0; i< sizeof(m_status_table)/sizeof(m_status_table[0]); ++i) {
            if (m_status_table[i].preStatus == stsPre && m_status_table[i].pointStatus == pointSts) {
                stsNow = m_status_table[i].nowStatus;
                edTp = m_status_table[i].edgeType;
                ADCDB_DEBUG(m_isShowDebugMsg, "match status[%4.4f], stsNow[%d], edTp[%d]", d, stsNow, edTp);
                if (edTp != EDGE_NOTFOUND) {
                    if (edTp == edgeType || edgeType == EDGE_BOTH) {
                        point = getPositionInRange(static_cast<uint32_t>(index), chSize);
                        frameStartPositions.push_back(make_pair(TRIGGER_FRAME_START, point));
                    }
                }
                break;
            }
        }
        stsPre = stsNow;
    }
    m_triggerStatus = stsNow;
    return frameStartPositions;
}

uint32_t AsyncCommand::getThresholdFirstLargeThan(vector<pair<uint16_t, uint32_t> > v, uint32_t d) {
    uint32_t ret = 0;
    for (auto& p : v) {
        if (p.second > d) {
            ret = p.second;
            break;
        }
    }
    return ret;
}

uint32_t AsyncCommand::getPositionInRange(uint32_t pos, uint32_t count) {
    uint32_t p = pos;
    return ((p == 0) ? ++p : (p == (count-1) ? --p : p));
}

vector<pair<uint16_t, uint32_t> > AsyncCommand::getTriggerMaskPoint(uint32_t dataCount, vector<pair<uint16_t, uint32_t> > thresholdPoint) {
    vector<pair<uint16_t, uint32_t> > ret;
    uint32_t currentPos = 0;
    uint32_t saveThresholdPos = 0;
    if (m_sendedPoints == m_tiggerMaskPoints || m_sendedPoints == 0) {
        currentPos = 0;
    } else {
        currentPos = m_tiggerMaskPoints - m_sendedPoints;
    }
    saveThresholdPos = getThresholdFirstLargeThan(thresholdPoint, currentPos);
    do {
        if (saveThresholdPos == 0) {
            break;
        }
        ret.push_back(make_pair(TRIGGER_FRAME_START, saveThresholdPos));
        currentPos = saveThresholdPos + m_tiggerMaskPoints;
        saveThresholdPos = getThresholdFirstLargeThan(thresholdPoint, currentPos);
    } while (true);
    if (ret.size() > 0) {
        m_sendedPoints = dataCount - ret.back().second;
    } else {
        uint32_t count = m_sendedPoints + dataCount;
        m_sendedPoints = (count < m_tiggerMaskPoints) ? count : m_tiggerMaskPoints;
    }
    return ret;
}
