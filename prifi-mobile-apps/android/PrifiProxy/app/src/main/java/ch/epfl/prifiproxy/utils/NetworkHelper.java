package ch.epfl.prifiproxy.utils;

import android.util.Log;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.net.SocketAddress;
import java.util.regex.Pattern;

public class NetworkHelper {

    private static final String TAG = NetworkHelper.class.getName();

    private static final Pattern IPV4_PATTERN = Pattern.compile(
            "^(25[0-5]|2[0-4]\\d|[0-1]?\\d?\\d)(\\.(25[0-5]|2[0-4]\\d|[0-1]?\\d?\\d)){3}$");

    private static final Pattern PORT_PATTERN = Pattern.compile(
            "^(6553[0-5]|655[0-2]\\d|65[0-4]\\d\\d|6[0-4]\\d{3}|[1-5]\\d{4}|[2-9]\\d{3}|1[1-9]\\d{2}|10[3-9]\\d|102[4-9])$");


    public static boolean isHostReachable(String serverAddress, int serverTcpPort, int timeout){
        boolean connected = false;
        Socket socket;

        try {
            socket = new Socket();
            SocketAddress socketAddress = new InetSocketAddress(serverAddress, serverTcpPort);
            socket.connect(socketAddress, timeout);
            if (socket.isConnected()) {
                connected = true;
                socket.close();
            }
        } catch (IOException e) {
            Log.i(TAG, "Cannot connect to the host " + serverAddress + ":" + String.valueOf(serverTcpPort));
        }

        return connected;
    }

    public static boolean isValidIpv4Address(String address) {
        return IPV4_PATTERN.matcher(address).matches();
    }

    public static boolean isValidPort(String port) {
        return PORT_PATTERN.matcher(port).matches();
    }

}
