from . import quantizer_body
import numpy as np
class Quantizer(object):
    def __init__(self, x_min:int, x_max:int, x_count:int, y_min:float, y_max:float, y_count:int):
        self.x_min   = x_min  
        self.x_max   = x_max  
        self.x_count = x_count
        self.y_min   = y_min  
        self.y_max   = y_max  
        self.y_count = y_count

    def quantize(self, input:np.array, output:np.array=None):
        if output is None:
            output = np.zeros(self.x_count*self.y_count)
        quantizer_body.quantize_body(self.x_min, self.x_max, self.x_count, self.y_min, self.y_max, self.y_count, input, output)
        return output
