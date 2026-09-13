# Balance baseline

Early-combat difficulty curve. Player is fixed to the worst case of un-upgraded `ash` + `bare_hands`.
power ratio is the ratio of how fast the player kills to how fast the player dies; 1.0 is even, higher is easier.
Target power ratio is day1=2.5 to day20=1.3, band +/-0.3. The target is a design hypothesis to revisit by playing.

## 洞窟 (cave)

| day | danger | power ratio | target | in band |
|---:|---:|---:|---:|:--:|
| 1 | 1 | 2.46 | 2.50 | in |
| 2 | 1 | 2.46 | 2.44 | in |
| 3 | 2 | 2.43 | 2.37 | in |
| 4 | 2 | 2.43 | 2.31 | in |
| 5 | 2 | 2.43 | 2.25 | in |
| 6 | 3 | 2.39 | 2.18 | in |
| 7 | 3 | 2.39 | 2.12 | in |
| 8 | 3 | 2.39 | 2.06 | out |
| 9 | 4 | 2.39 | 1.99 | out |
| 10 | 4 | 2.39 | 1.93 | out |
| 11 | 4 | 2.39 | 1.87 | out |
| 12 | 5 | 2.39 | 1.81 | out |
| 13 | 5 | 2.39 | 1.74 | out |
| 14 | 5 | 2.39 | 1.68 | out |
| 15 | 6 | 2.25 | 1.62 | out |
| 16 | 6 | 2.25 | 1.55 | out |
| 17 | 6 | 2.25 | 1.49 | out |
| 18 | 7 | 2.08 | 1.43 | out |
| 19 | 7 | 2.08 | 1.36 | out |
| 20 | 7 | 2.08 | 1.30 | out |
| 21 | 8 | 2.06 | 1.30 | out |

## 森 (forest)

| day | danger | power ratio | target | in band |
|---:|---:|---:|---:|:--:|
| 1 | 1 | 2.84 | 2.50 | out |
| 2 | 1 | 2.84 | 2.44 | out |
| 3 | 2 | 2.73 | 2.37 | out |
| 4 | 2 | 2.73 | 2.31 | out |
| 5 | 2 | 2.73 | 2.25 | out |
| 6 | 3 | 2.61 | 2.18 | out |
| 7 | 3 | 2.61 | 2.12 | out |
| 8 | 3 | 2.61 | 2.06 | out |
| 9 | 4 | 2.54 | 1.99 | out |
| 10 | 4 | 2.54 | 1.93 | out |
| 11 | 4 | 2.54 | 1.87 | out |
| 12 | 5 | 2.44 | 1.81 | out |
| 13 | 5 | 2.44 | 1.74 | out |
| 14 | 5 | 2.44 | 1.68 | out |
| 15 | 6 | 2.40 | 1.62 | out |
| 16 | 6 | 2.40 | 1.55 | out |
| 17 | 6 | 2.40 | 1.49 | out |
| 18 | 7 | 2.32 | 1.43 | out |
| 19 | 7 | 2.32 | 1.36 | out |
| 20 | 7 | 2.32 | 1.30 | out |
| 21 | 8 | 2.28 | 1.30 | out |

## 廃墟 (ruins_area)

| day | danger | power ratio | target | in band |
|---:|---:|---:|---:|:--:|
| 1 | 1 | 2.59 | 2.50 | in |
| 2 | 1 | 2.59 | 2.44 | in |
| 3 | 2 | 2.58 | 2.37 | in |
| 4 | 2 | 2.58 | 2.31 | in |
| 5 | 2 | 2.58 | 2.25 | out |
| 6 | 3 | 2.42 | 2.18 | in |
| 7 | 3 | 2.42 | 2.12 | out |
| 8 | 3 | 2.42 | 2.06 | out |
| 9 | 4 | 2.24 | 1.99 | in |
| 10 | 4 | 2.24 | 1.93 | out |
| 11 | 4 | 2.24 | 1.87 | out |
| 12 | 5 | 1.97 | 1.81 | in |
| 13 | 5 | 1.97 | 1.74 | in |
| 14 | 5 | 1.97 | 1.68 | in |
| 15 | 6 | 1.97 | 1.62 | out |
| 16 | 6 | 1.97 | 1.55 | out |
| 17 | 6 | 1.97 | 1.49 | out |
| 18 | 7 | 1.88 | 1.43 | out |
| 19 | 7 | 1.88 | 1.36 | out |
| 20 | 7 | 1.88 | 1.30 | out |
| 21 | 8 | 1.88 | 1.30 | out |

## survival pressure

Time to starve without food, and time to hypothermia by effective temperature (ambient + insulation).

| metric | value |
|---|---:|
| days until starving (hunger < 33%) | 0.67 |
| days until hunger empty | 1.00 |

| effective temp (C) | turns to hypothermia |
|---:|---:|
| -20 | 10 |
| -10 | 10 |
| 0 | 10 |
| 5 | 20 |
| 10 | 20 |
| 15 | 0 |

## sensitivity (ruins day20 power ratio, +10% each knob)

| knob | base | +10% | change |
|---|---:|---:|---:|
| bite | 1.88 | 1.84 | -2.0% |
| cleaver | 1.88 | 1.88 | -0.4% |
| flame_attack | 1.88 | 1.88 | -0.1% |
| bare_hands | 1.88 | 1.88 | +0.0% |

