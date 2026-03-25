# Windows-proxy-for-wsl
Windows proxy to redirect socket to WSL. The main reason to use that is because my mirror mode won't work and the NAT mode is troublesome, because we have to keep configuring everytime the wsl ip changes


How to make it work: 

just put both files on the same folder and double click on wsl_proxy.bat

You should allow these ports on windows firewall

On NAT MODE, you can run your application on wsl on port 3000 (example) and then after clicking on wsl_proxy.bat type 3000 and enter.

It accepts multiple ports like: 3000 8000

On mirror mode, you should open your wsl on port 3001, open the port 3000 on windows, then when you run the .bat, type 3000:3001 and it will work.
