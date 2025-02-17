import numpy as np

mat = np.random.rand(9000,9000)
print("Matrix with", mat.size, "cells:\n", mat)

mean = mat.mean()
print("has mean value:", mean)

det = np.linalg.det(mat)
print("has determinant:", det)

exp = np.linalg.matrix_power(mat, 10)
print("exponentiated:\n", exp)

#U, s, Vt = np.linalg.svd(mat)
#print("decomposition:", s)

#mat
