import pandas as pd
import numpy as np

pat = pd.read_csv("patient_locations_sm.csv.gz")
prov = pd.read_csv("provider_locations_sm.csv.gz")

for d in pat, prov:
    d["Match"] = d["Match"].replace({"T": 1, "F": 0})

out = open("rfid_summaries.txt", "w")
out.write("```\n")

out.write("%d total patient minutes\n" % pat.shape[0])
out.write("%d total provider minutes\n\n" % prov.shape[0])

out.write("%d distinct CSN values\n" % pat.CSN.unique().size)
out.write("%d distinct provider id's\n\n" % prov.UMid.unique().size)

# Distribution of minutes per room, for patients
a = pat["Room_HMM"].value_counts()
a = pd.DataFrame(a)
a.columns = ["Minutes"]
a["Percentage"] = 100 * a.Minutes / a.Minutes.sum()
out.write("Total patient minutes per location:\n%s\n\n" % a.to_string(float_format="%.1f"))

# Distribution of minutes per room, for providers
a = prov["Room_HMM"].value_counts()
a = pd.DataFrame(a)
a.columns = ["Minutes"]
a["Percentage"] = 100 * a.Minutes / a.Minutes.sum()
out.write("Total provider minutes per location:\n%s\n\n" % a.to_string(float_format="%.1f"))

# Distribution
a = prov.groupby("Provider").size()
a = pd.DataFrame(a)
a.columns = ["Minutes"]
a["Percentage"] = 100 * a.Minutes / a.Minutes.sum()
out.write("Total provider minutes by provider type:\n%s\n\n" % a.to_string(float_format="%.1f"))

# Average number of patients per room, by room.
xx = pat.groupby(["Room_HMM", "Time"]).size()
xx = xx.reset_index()
xx = xx.groupby("Room_HMM").agg(np.mean)
xx.columns = ["Patients"]
out.write("Average number of patients per room given at least 1 patient is present:\n")
out.write(xx.to_string(float_format="%.3f") + "\n\n")

# Average number of providers per room, by room.
xx = prov.groupby(["Room_HMM", "Time"]).size()
xx = xx.reset_index()
xx = xx.groupby("Room_HMM").agg(np.mean)
xx.columns = ["Providers"]
out.write("Average number of providers per room given at least 1 patient is present:\n")
out.write(xx.to_string(float_format="%.3f") + "\n\n")

# Distribution of percentage of minutes with provider in room, for patients
a = pat.groupby("CSN").agg({"Match": lambda x: 100*np.mean(x)})
out.write("Distribution of provider-in-room percentages, per appointment:\n")
out.write("%s\n\n" % a.describe().to_string(float_format="%.1f"))

# Distribution of percentage of minutes with patient in room, for providers
a = prov.groupby("UMid").agg({"Match": lambda x: 100*np.mean(x)})
out.write("Distribution of patient-in-room percentages, per provider:\n")
out.write("%s\n\n" % a.describe().to_string(float_format="%.1f"))

# Percentage of patient-in-room time, per provider type
a = prov.groupby("Provider").agg({"Match": lambda x : 100 * np.mean(x)})
out.write("Patient-in-room percentage, by provider type:\n")
out.write(a.to_string(float_format="%.1f") + "\n\n")

# Distribution of minutes per room, for providers
a = prov.groupby("Room_HMM").agg({"Match": lambda x: 100 * np.mean(x)})
a = pd.DataFrame(a)
out.write("Provider-in-room percentage, by room:\n%s\n\n" % a.to_string(float_format="%.1f"))

out.write("```\n")
out.close()