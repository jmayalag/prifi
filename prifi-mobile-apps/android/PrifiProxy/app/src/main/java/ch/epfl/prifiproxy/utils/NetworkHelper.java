package ch.epfl.prifiproxy.utils;

import android.util.Log;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.net.SocketAddress;
import java.util.regex.Pattern;

/**
 * This class provides useful network-related functions.
 */
public class NetworkHelper {

    private static final String TAG = NetworkHelper.class.getName();

    private static final Pattern IPV4_PATTERN = Pattern.compile(
            "^(25[0-5]|2[0-4]\\d|[0-1]?\\d?\\d)(\\.(25[0-5]|2[0-4]\\d|[0-1]?\\d?\\d)){3}$");

    private static final Pattern PORT_PATTERN = Pattern.compile(
            "^(6553[0-5]|655[0-2]\\d|65[0-4]\\d\\d|6[0-4]\\d{3}|[1-5]\\d{4}|[2-9]\\d{3}|1[1-9]\\d{2}|10[3-9]\\d|102[4-9])$");


    /**
     * This function tries to establish a TCP connexion to the given host:port
     * in order to check if the server is available or not.
     * @param serverAddress host
     * @param serverTcpPort port
     * @param timeout timeout
     * @return true if the host is reachable, false if the host is not reachable.
     */
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

    /**
     * Check if the given string is a valid IPv4 address.
     * @param address An IPv4 address to check
     * @return true if the address is a valid IPv4 address, otherwise false
     */
    public static boolean isValidIpv4Address(String address) {
        return IPV4_PATTERN.matcher(address).matches();
    }

    /**
     * Check if the given port is a valid port.
     * @param port A port to check
     * @return true if the port is a valid port, otherwise false
     */
    public static boolean isValidPort(String port) {
        return PORT_PATTERN.matcher(port).matches();
    }

}
