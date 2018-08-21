#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <algorithm>
#include <iterator>     // std::back_inserter
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
using namespace std;
uint32_t m_sendedPoints = 0;
uint64_t m_tiggerMaskPoints = 200;
enum {
    TRIGGER_FRAME_MID = 0,
    TRIGGER_FRAME_START = 1,
    TRIGGER_FRAME_END = 2,
};


uint32_t getThresholdFirstLargeThan(vector<pair<uint16_t, uint32_t> > v, uint32_t d) {
    uint32_t ret = 0;
    for (auto& p : v) {
        if (p.second > d) {
            ret = p.second;
            break;
        }
    }
    return ret;
}

uint32_t getPositionInRange(uint32_t pos, uint32_t count) {
    uint32_t p = pos;
    if (p == 0) {
        p++;
    } else
        if (p == (count-1)) {
            p--;
        } else {
            /*do nothing*/
        }
    return p;
}

vector<pair<uint16_t, uint32_t> > getTriggerMaskPoint(uint32_t dataCount, vector<pair<uint16_t, uint32_t> > thresholdPoint) {
#if 0
    vector<pair<uint16_t, uint32_t> > ret;
    uint32_t currentPos = 0;
    bool isIgnoreThreshold = false;
    if (m_sendedPoints > 0 && m_sendedPoints < m_tiggerMaskPoints) {
        currentPos = m_tiggerMaskPoints - m_sendedPoints - 1;
        if (currentPos < dataCount) {
            ret.push_back(make_pair(TRIGGER_FRAME_END, currentPos));
            m_sendedPoints = m_tiggerMaskPoints;
        }
        else {
            m_sendedPoints += dataCount;
            isIgnoreThreshold = true;
        }
    }
   if (isIgnoreThreshold == false && thresholdPoint.size() > 0) {
        uint32_t startPos = thresholdPoint.at(0).second;
        for (auto& pos : thresholdPoint) {
            if (pos.second < currentPos) {
                startPos = getThresholdFirstLargeThan(thresholdPoint, currentPos);
                continue;
            }
            if (m_sendedPoints == m_tiggerMaskPoints || m_sendedPoints == 0) {
                if (pos.second - startPos == 0) {
                    startPos = getPositionInRange(startPos, dataCount);
                    ret.push_back(make_pair(TRIGGER_FRAME_START, startPos));
                    startPos = pos.second;
                    currentPos = startPos;
                    m_sendedPoints = 1;
                }
            } else {
                if (pos.second - startPos >= m_tiggerMaskPoints) {
                    uint32_t endPoint = startPos + m_tiggerMaskPoints;
                    endPoint = getPositionInRange(endPoint, dataCount);
                    ret.push_back(make_pair(TRIGGER_FRAME_END, endPoint));
                    startPos = (pos.second - startPos == m_tiggerMaskPoints)? pos.second + 1 : pos.second;
                    currentPos = startPos;
                    startPos = (startPos == dataCount - 1) ? startPos - 1 : startPos;
                    endPoint = getPositionInRange(startPos, dataCount);
                    ret.push_back(make_pair(TRIGGER_FRAME_START, startPos));
                    m_sendedPoints = (endPoint < dataCount) ?  1 : dataCount - startPos;
                }
                else {
                    m_sendedPoints = pos.second - startPos;
                }
            }
        }
    }
    return ret;
#else
    vector<pair<uint16_t, uint32_t> > ret;
    uint32_t currentPos = 0;
    uint32_t saveThresholdPos = 0;
    if (m_sendedPoints == m_tiggerMaskPoints || m_sendedPoints == 0) {
        currentPos = 0;
    }
    else {
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
#endif
}

int main(int argc, char const* argv[])
{
        /*
        vector<int> vec(100);
        for (size_t i = 0; i < vec.size(); ++i) {
            vec[i] = i;
        }

        std::random_shuffle (vec.begin(), vec.end());

        // std::*_element は，イテレーターを返すので '*' で値を取得する
        int min = *std::min_element(vec.begin(), vec.end());
        int max = *std::max_element(vec.begin(), vec.end());
        
        cout << min << endl;
        cout << max << endl;
        
        cout << "======" << endl;
        */
        
        /*
        vector<int> vdata, vindex;
        map<int, vector<int> > mmp;
        for(int i = 0;i<50;i++) {
           vdata.push_back(i);
        }
        vindex.push_back(10);
        vindex.push_back(20);
        vindex.push_back(30);
        vindex.push_back(40);

        int j = 0,prePos = 0;
        vector<int> vec;
        for (; j < vindex.size(); ++j) {
            uint32_t point = vindex.at(j);
            vec.clear();
            copy(vdata.begin()+prePos, vdata.begin()+point, back_inserter(vec));
            mmp.insert(make_pair(j, vec));
            prePos = point;
        }

        if (prePos < vdata.size()) {
            vec.clear();
            copy(vdata.begin()+prePos, vdata.end(), back_inserter(vec));
            mmp.insert(make_pair(j, vec));
        }

        for (auto& v : mmp) {
           cout << "======" << v.first << endl;
           std::for_each(v.second.begin(), v.second.end(), [&](const int d) {
                cout << d << endl;
           });
        }
        */
        
       
        vector<pair<uint16_t, uint32_t> > thresholdPointVec;
        thresholdPointVec.push_back(make_pair(1, 1));
        thresholdPointVec.push_back(make_pair(1, 50));
        thresholdPointVec.push_back(make_pair(1, 100));
        thresholdPointVec.push_back(make_pair(1, 200));
        thresholdPointVec.push_back(make_pair(1, 500));
        thresholdPointVec.push_back(make_pair(1, 1000));
        thresholdPointVec.push_back(make_pair(1, 1980));
        
        m_sendedPoints = 80;
        m_tiggerMaskPoints = 3000;
        for (int i=0; i<2; ++i) {
            vector<pair<uint16_t, uint32_t> > triggerPointVec = getTriggerMaskPoint(2000, thresholdPointVec);
             cout << "m_sendedPoints : " << m_sendedPoints <<  endl;
            for (auto& v : triggerPointVec) {
                cout << v.first << ", " << v.second <<  endl;
            }
        }        

        return 0;
}
