from __future__ import print_function
import pyethereum
t = pyethereum.tester
s = t.state()
# Create currencies
c1 = s.contract('subcurrency.se')
print("First currency: %s" % c1)
c2 = s.contract('subcurrency.se')
print("First currency: %s" % c2)
# Allocate units
s.send(t.k0, c1, 0, [t.a0, 1000, 0])
s.send(t.k0, c1, 0, [t.a1, 1000, 0])
s.send(t.k0, c2, 0, [t.a2, 1000000, 0])
s.send(t.k0, c2, 0, [t.a3, 1000000, 0])
print("Allocated units")
# Market
m = s.contract('market.se')
s.send(t.k0, m, 0, [c1, c2])
# Place orders
s.send(t.k0, c1, 0, [m, 1000])
s.send(t.k0, m, 0, [0, 1200])
s.send(t.k1, c1, 0, [m, 1000])
s.send(t.k1, m, 0, [0, 1400])
s.send(t.k2, c2, 0, [m, 1000000])
s.send(t.k2, m, 0, [1, 800])
s.send(t.k3, c2, 0, [m, 1000000])
s.send(t.k3, m, 0, [1, 600])
print("Orders placed")
# Next epoch and ping
s.mine(100)
print("Mined 100")
s.send(t.k0, m, 0, [])
print("Updating")
# Check
assert s.send(t.k0, c2, 0, [t.a0]) == [800000]
assert s.send(t.k0, c2, 0, [t.a1]) == [600000]
assert s.send(t.k0, c1, 0, [t.a2]) == [833]
assert s.send(t.k0, c1, 0, [t.a3]) == [714]
print("Balance checks passed")
