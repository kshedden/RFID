import numpy as np
import matplotlib
matplotlib.use('Agg')
import matplotlib.patches as patches
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from matplotlib.font_manager import FontProperties
from matplotlib.backends.backend_pdf import PdfPages
import pandas as pd
import matplotlib.dates as mdates
from matplotlib.transforms import blended_transform_factory

provider = True

if provider:
    df = pd.read_csv("provider_locations_s.csv.gz")
    idvar = "UMid"
else:
    df = pd.read_csv("patient_locations_s.csv.gz")
    idvar = "CSN"

font0 = FontProperties()
font0.set_family("monospace")
font0.set_size(9)

df["Time"] = pd.to_datetime(df.Time)

df["Day"] = df.Time.dt.dayofyear

# Discover the time step
u = pd.Series(pd.to_datetime(df.Time.unique()))
u = u.sort_values()
u = u.diff()
u = u.loc[u != pd.to_timedelta(0)]
timestep = u.min()

clr = plt.get_cmap("Set3").colors
colors = {"Exam1": clr[0],
          "Exam2": clr[0],
          "Exam3": clr[0],
          "Exam4": clr[0],
          "Exam5": clr[0],
          "Exam6": clr[0],
          "Exam7": clr[0],
          "Exam8": clr[0],
          "Exam9": clr[0],
          "Exam10": clr[0],
          "Exam11": clr[0],
          "Exam12": clr[0],
          "Field1": clr[1],
          "Field2": clr[1],
          "Field3": clr[1],
          "Field4": clr[1],
          "Field5": clr[1],
          "IOLMaster": clr[2],
          "Lensometer": clr[3],
          "Admin": clr[4],
          "Checkout": clr[5],
          "IPW9": clr[6],
          "IPW2": clr[6],
          "Treatment": clr[7],
          "NoSignal": clr[8],
          "CheckoutReturn": clr[5]}

if provider:
    pdf = PdfPages("provider_locs.pdf")
else:
    pdf = PdfPages("patient_locs.pdf")

iyv = []

for day, f in df.groupby("Day"):

    print(day)

    if f.Time.iloc[0].weekday() in (5, 6):
    	continue

    if f.shape[0] < 100:
        continue

    # Vertical position on the plot
    iy = 0

    plt.clf()

    vl = set([])
    plt.figure(figsize=(10, 6))
    ax = plt.axes([0.1, 0.1, 0.7, 0.8])
    trans = blended_transform_factory(ax.transAxes, ax.transData)
    handles = []
    labels = []
    for csn, gx in f.groupby(idvar):

        # DEBUG
        print(csn)

        if provider:
        	mn = f.Time.min()
        	pt = gx.Provider.iloc[0]
        	plt.text(-0.13, iy, pt, ha='left', color='grey',
        		fontproperties=font0, transform=trans)

        for jr in 1, 2:

            if jr == 1:
                room = "Room%d" % jr
            else:
                room = "Room_HMM"

            signal = "Signal%d" % jr
            rw = {1: 3, 2: 1}[jr]

            g = gx.copy()
            while g.shape[0] > 0:

                if g[room].unique().size == 1:
                    h = g
                    g = g.loc[[], :]
                else:
                    for i in range(1, g.shape[0]):
                        if g[room].iloc[i] != g[room].iloc[i-1]:
                            h = g.iloc[0:i, :]
                            g = g.iloc[i:, :]
                            break
                rmx = h[room].iloc[0]

                if pd.isnull(rmx):
                    continue

                # Provider
                if csn == 0:
                    continue

                h = h.sort_values(by="Time")

                #if jr == 2 and h["Signal2"].max() < 0.5 * h["Signal1"].max():
                #    continue

                mn = h.Time.min().to_pydatetime()
                mx = h.Time.max().to_pydatetime() + timestep

                rect = patches.Rectangle([mdates.date2num(mn), iy], mdates.date2num(mx)-mdates.date2num(mn),
                                         rw, facecolor=colors[rmx], edgecolor=colors[rmx], lw=0.8)
                rect.set_joinstyle("miter")
                rect.set_capstyle("projecting")
                plt.gca().add_patch(rect)
                if rmx not in labels:
                    handles.append(patches.Rectangle([0, 0], 4, 2, facecolor=colors[rmx], edgecolor="none"))
                    labels.append(rmx)

                # Transitions between places with the same color
                if g.shape[0] > 0 and colors[rmx] == colors[g[room].iloc[0]]:
                    plt.plot([mx, mx], [iy, iy+rw], '-', color='black', lw=0.1)

            iy += 1.5*rw

        iy += 2 # Additional space between pairs

    iyv.append(iy)

    tm = h.Time.min()
    ti = pd.to_datetime("%d-%d-%d 07:02:00" % (tm.year, tm.month, tm.day))

    # Display the date as the title
    ca = f.Time.iloc[0].timetuple()[0:3]
    ca = [str(x) for x in ca]
    ca = "-".join(ca)
    plt.title(ca)

    mp = {x: y for x, y in zip(labels, handles)}
    labels = list(mp.keys())
    labels.sort()
    handles = [mp[k] for k in labels]
    leg = plt.figlegend(handles, labels, "center right")
    leg.draw_frame(False)

    # Limit the display to 7am-7pm
    t0 = f.Time.min().to_pydatetime()
    t0 = t0.replace(hour=7, minute=0)
    t1 = f.Time.max().to_pydatetime()
    t1 = t1.replace(hour=19, minute=0)
    plt.xlim(t0, t1)

    plt.gca().set_yticks([])
    plt.gca().xaxis.set_major_locator(mdates.MinuteLocator(interval=120))
    plt.gca().xaxis.set_major_formatter(mdates.DateFormatter('%H:%M'))

    plt.ylim(-5, 365)

    pdf.savefig()

    #DEBUG
    if day >= 30:
        break

pdf.close()
