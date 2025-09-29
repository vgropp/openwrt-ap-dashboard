import React from "react";

export const SignalBars: React.FC<{ signal: number }> = ({ signal }) => {
  // Map dBm to 0–4 bars (roughly like Wi-Fi icons)
  let bars = 0;
  if (signal > -50) bars = 4;
  else if (signal > -60) bars = 3;
  else if (signal > -70) bars = 2;
  else if (signal > -80) bars = 1;
  else bars = 0;

  return (
    <div className="flex space-x-0.5 items-end h-5">
      {[1, 2, 3, 4].map((i) => (
        <div
          key={i}
          className={`w-1.5 rounded-sm ${
            i <= bars ? "bg-blue-500" : "bg-gray-300"
          }`}
          style={{ height: `${i * 5}px` }}
        />
      ))}
    </div>
  );
};
