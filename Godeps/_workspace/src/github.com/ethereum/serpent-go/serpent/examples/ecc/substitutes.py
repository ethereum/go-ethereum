import bitcoin as b
import math
import sys


def signed(o):
    return map(lambda x: x - 2**256 if x >= 2**255 else x, o)


def hamming_weight(n):
    return len([x for x in b.encode(n, 2) if x == '1'])


def binary_length(n):
    return len(b.encode(n, 2))


def jacobian_mul_substitute(A, B, C, D, N):
    if A == 0 and C == 0 or (N % b.N) == 0:
        return {"gas": 86, "output": [0, 1, 0, 1]}
    else:
        output = b.jordan_multiply(((A, B), (C, D)), N)
        return {
            "gas": 35262 + 95 * binary_length(N % b.N) + 355 * hamming_weight(N % b.N),
            "output": signed(list(output[0]) + list(output[1]))
        }


def jacobian_add_substitute(A, B, C, D, E, F, G, H):
    if A == 0 or E == 0:
        gas = 149
    elif (A * F - B * E) % b.P == 0:
        if (C * H - D * G) % b.P == 0:
            gas = 442
        else:
            gas = 177
    else:
        gas = 301
    output = b.jordan_add(((A, B), (C, D)), ((E, F), (G, H)))
    return {
        "gas": gas,
        "output": signed(list(output[0]) + list(output[1]))
    }


def modexp_substitute(base, exp, mod):
    return {
        "gas": 5150,
        "output": signed([pow(base, exp, mod) if mod > 0 else 0])
    }


def ecrecover_substitute(z, v, r, s):
    P, A, B, N, Gx, Gy = b.P, b.A, b.B, b.N, b.Gx, b.Gy
    x = r
    beta = pow(x*x*x+A*x+B, (P + 1) / 4, P)
    BETA_PREMIUM = modexp_substitute(x, (P + 1) / 4, P)["gas"]
    y = beta if v % 2 ^ beta % 2 else (P - beta)
    Gz = b.jordan_multiply(((Gx, 1), (Gy, 1)), (N - z) % N)
    GZ_PREMIUM = jacobian_mul_substitute(Gx, 1, Gy, 1, (N - z) % N)["gas"]
    XY = b.jordan_multiply(((x, 1), (y, 1)), s)
    XY_PREMIUM = jacobian_mul_substitute(x, 1, y, 1, s % N)["gas"]
    Qr = b.jordan_add(Gz, XY)
    QR_PREMIUM = jacobian_add_substitute(Gz[0][0], Gz[0][1], Gz[1][0], Gz[1][1],
                                         XY[0][0], XY[0][1], XY[1][0], XY[1][1]
                                         )["gas"]
    Q = b.jordan_multiply(Qr, pow(r, N - 2, N))
    Q_PREMIUM = jacobian_mul_substitute(Qr[0][0], Qr[0][1], Qr[1][0], Qr[1][1],
                                        pow(r, N - 2, N))["gas"]
    R_PREMIUM = modexp_substitute(r, N - 2, N)["gas"]
    OX_PREMIUM = modexp_substitute(Q[0][1], P - 2, P)["gas"]
    OY_PREMIUM = modexp_substitute(Q[1][1], P - 2, P)["gas"]
    Q = b.from_jordan(Q)
    return {
        "gas": 991 + BETA_PREMIUM + GZ_PREMIUM + XY_PREMIUM + QR_PREMIUM +
        Q_PREMIUM + R_PREMIUM + OX_PREMIUM + OY_PREMIUM,
        "output": signed(Q)
    }
