import numpy as np
import os
import sys
import socket
import argparse
import models
import time
import struct
import select
from quantizer import Quantizer

parser = argparse.ArgumentParser()
parser.add_argument('--inputs', type = int, default = 784)
parser.add_argument('--units', type = int, default = 1024)

# Quantization parameters
parser.add_argument('--quantize-x-count', dest='quantize_x_count', type=int)
parser.add_argument('--quantize-y-min', dest='quantize_y_min', type=float)
parser.add_argument('--quantize-y-max', dest='quantize_y_max', type=float)
parser.add_argument('--quantize-y-count', dest='quantize_y_count', type=int)

parser.add_argument('--activation', choices = ['sigmoid', 'relu', 'linear'], default = 'sigmoid')
parser.add_argument('--loss', choices = ['mean_squared_error', 'l1_error'], default = 'mean_squared_error')
parser.add_argument('--port', type = int, default = 9999)

RESULT_PACKET_MAGIC = 0x0115eaa0	# OliveAI
RESULT_PACKET_TYPE_TRAINING_PROGRESS = 0
RESULT_PACKET_TYPE_TRAINING_DONE     = 1
RESULT_PACKET_TYPE_INFERENCE_RESULT  = 2

def recvall(sock, msglen):
	recvlen = 0
	msg = bytes()
	while recvlen < msglen:
		part = sock.recv(msglen - recvlen)
		if len(part) == 0:
			raise RuntimeError('Connection closed')
		msg += part
		recvlen += len(part)
	return msg

def main(args):
	print("Initializing OS-ELM model with following parameters:")
	print("\tinputs:     {0}".format(args.inputs))
	print("\tunits:      {0}".format(args.units))
	print("\tactivation: {0}".format(args.activation))
	print("\tloss:       {0}".format(args.loss))
	print("\tport:       {0}".format(args.port))
	
	model_inputs = args.inputs
	if args.quantize_x_count is not None:
		print("\tquantize X count: {0}".format(args.quantize_x_count))
		print("\tquantize Y count: {0}".format(args.quantize_y_count))
		print("\tquantize Y min:   {0}".format(args.quantize_y_min))
		print("\tquantize Y max:   {0}".format(args.quantize_y_max))
		quantizer = Quantizer(	x_min=0, x_max=args.inputs-1, x_count=args.quantize_x_count,
								y_min=args.quantize_y_min, y_max=args.quantize_y_max, y_count=args.quantize_y_count)
		model_inputs = args.quantize_x_count*args.quantize_y_count
		quantizer_outputs = np.zeros(model_inputs, dtype=np.float32)
	else:
		quantizer = None
	
	os_elm = models.OS_ELM(
		inputs = model_inputs,
		units = args.units,
		outputs = model_inputs,
		activation = args.activation,
		loss = args.loss)

	border = args.units #int(args.units * 1.1)

	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	s.bind(('', args.port))
	
	print("Python OS-ELM server listening at port {0}".format(args.port))
	s.listen(10)

	count = 0
	x_data = bytes() if quantizer is None else np.zeros((args.units * model_inputs), dtype=np.float32)

	c, addr = s.accept()
	s.close()
	last_training_progress_time = time.monotonic()
	print("Connected from {0}".format(addr))
	while True:
		try:
			readable, _, error = select.select((c,), (c,), (c,), 5)
			if len(readable) > 0:
				if count < border:
					msg = recvall(c, args.inputs * 4)
					#print('Received data... {0} {1}'.format(count, len(msg)))
					
					if quantizer is not None:
						input = np.frombuffer(msg, dtype = np.float32)
						quantizer.quantize(input=input, output=x_data[count*model_inputs:(count+1)*model_inputs])
					else:
						x_data += msg
					
					test_loss = -1
					now = time.monotonic()
					if now - last_training_progress_time >= 1:
						last_training_progress_time = now
						packed = struct.pack("<iii", RESULT_PACKET_MAGIC, RESULT_PACKET_TYPE_TRAINING_PROGRESS, count)
						c.sendall(packed)

				if count >= border:
					
					#print('Receiving a data...')
					msg = recvall(c, args.inputs * 4)
					#print("Received length={0}".format(len(msg)))
					x_data = np.frombuffer(msg, dtype = np.float32)
					#print("Received width={0}".format(len(x_data)))
					x_data = x_data.reshape(-1, args.inputs)
					if quantizer is not None:
						x_data = quantizer.quantize(input=x_data[0], output=quantizer_outputs).reshape((-1, model_inputs))
					
					#print('Evaluating the loss...')
					test_loss = os_elm.compute_loss(x_data, x_data)
					packed = struct.pack("<iif", RESULT_PACKET_MAGIC, RESULT_PACKET_TYPE_INFERENCE_RESULT, test_loss)
					c.sendall(packed)
					#print('loss {0}'.format(test_loss))
					#print('Training the data...')
					#start = time.time()
					os_elm.seq_train(x_data, x_data)
					#elapsed_time = time.time() - start
					#print('Elapsed time %f [sec]' % elapsed_time)

				if count == border - 1:
					print('Finalizing sequential training... length={0}'.format(len(x_data)))
					if quantizer is None:
						x_data = np.frombuffer(x_data, dtype = np.float32)
						x_data = x_data.reshape(-1, args.inputs)
					else:
						x_data = x_data.reshape(-1, model_inputs)
					
					#print(x_data.shape)
					
					os_elm.init_train(x_data, x_data)
					print('Sequential training done...')
					packed = struct.pack("<iii", RESULT_PACKET_MAGIC, RESULT_PACKET_TYPE_TRAINING_DONE, border)
					c.sendall(packed)

				count += 1
			elif len(error) > 0:
				print("Socket error")
				c.shutdown(2)
				c.close()
				break
		except RuntimeError:
			print("Disconnected")
			c.shutdown(2)
			c.close()
			break

if __name__ == '__main__':
	args = parser.parse_args()
	main(args)
