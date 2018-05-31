#include "quantizer.h"

void quantize(int x_min, int x_max, int x_count, float y_min, float y_max, int y_count, float* input, float* output)
{
    int box_width = (x_max - x_min) / x_count;
    float box_height = (y_max - y_min) / y_count;
    float box_area = box_width*box_height;
    float y_center = (y_max + y_min) / 2;
    for(int box_col = 0; box_col < x_count; box_col++) {
        int box_x = box_width * box_col + x_min;
        for(int box_row = 0; box_row < y_count; box_row++) {
            float box_y = box_height*box_row + y_min;
            float box_sum = 0;
            for( int ofs_x = 0; ofs_x < box_width; ofs_x++) {
                if(box_x+ofs_x >= x_max) break;
                
                float value = input[box_x+ofs_x];
                float value_high = value > y_center ? value : y_center;
                float value_low  = value < y_center ? value : y_center;
                float upper  = box_y + box_height;
                float bottom = box_y;
                value_high = value_high > upper ? upper : value_high < bottom ? bottom : value_high;
                value_low  = value_low  > upper ? upper : value_low  < bottom ? bottom : value_low;
                
                float area = value_high - value_low;
                box_sum += area;
            }
            output[box_col+box_row*x_count] = box_sum/box_area;
        }
    }
}
