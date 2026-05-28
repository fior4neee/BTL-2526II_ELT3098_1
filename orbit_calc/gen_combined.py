import sys
from pathlib import Path
sys.path.append(str(Path("C:/Users/twtvf/OneDrive/Documents/GitHub/BTL-2526II_ELT3098_1/orbit_calc")))

from coverage_calculator import WalkerDelta, VietnamCoverageAnalyzer

# Generate 306 internet satellites
c_internet = WalkerDelta(N=306, P=17, F=0, inclination_deg=53.0, altitude_km=500.0)
a_internet = VietnamCoverageAnalyzer(c_internet, 15.0, 86400.0, 60.0)
a_internet.export_tle("internet_306.tle")

# Generate 110 weather satellites
c_weather = WalkerDelta(N=110, P=10, F=0, inclination_deg=53.0, altitude_km=1000.0)
a_weather = VietnamCoverageAnalyzer(c_weather, 15.0, 86400.0, 60.0)
a_weather.export_tle("weather_110.tle")

# Combine them into one TLE file
with open("combined_416.tle", "w") as fout:
    with open("internet_306.tle", "r") as fin:
        fout.write(fin.read())
    with open("weather_110.tle", "r") as fin:
        # We need to change the names of the weather satellites so they don't clash
        lines = fin.readlines()
        for i in range(0, len(lines), 3):
            # Change "VNU-LEO" to "W-VNU-LEO" to signify weather
            name = lines[i].strip()
            if name.startswith("VNU-LEO"):
                name = name.replace("VNU-LEO", "W-VNU-LEO")
            fout.write(name + "\n")
            fout.write(lines[i+1])
            fout.write(lines[i+2])

print("Generated combined_416.tle")
