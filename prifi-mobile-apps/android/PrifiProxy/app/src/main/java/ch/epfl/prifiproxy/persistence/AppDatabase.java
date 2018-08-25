package ch.epfl.prifiproxy.persistence;

import android.arch.persistence.db.SupportSQLiteDatabase;
import android.arch.persistence.room.Database;
import android.arch.persistence.room.Room;
import android.arch.persistence.room.RoomDatabase;
import android.content.Context;
import android.os.AsyncTask;
import android.support.annotation.NonNull;
import android.util.Log;

import java.util.ArrayList;
import java.util.List;

import ch.epfl.prifiproxy.persistence.dao.ConfigurationDao;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationGroupDao;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

@Database(entities = {ConfigurationGroup.class, Configuration.class}, version = 1)
public abstract class AppDatabase extends RoomDatabase {
    public static final String DATABASE_NAME = "prifi_db";

    private static AppDatabase sInstance;

    public static AppDatabase getDatabase(Context context) {
        if (sInstance == null) {
            synchronized (AppDatabase.class) {
                if (sInstance == null) {
                    sInstance = Room.databaseBuilder(context.getApplicationContext(),
                            AppDatabase.class, DATABASE_NAME)
                            .addCallback(sRoomDatabaseCallback)
                            .build();
                }
            }
        }
        return sInstance;
    }

    public abstract ConfigurationDao configurationDao();

    public abstract ConfigurationGroupDao configurationGroupDao();

    private static RoomDatabase.Callback sRoomDatabaseCallback = new RoomDatabase.Callback() {
        @Override
        public void onOpen(@NonNull SupportSQLiteDatabase db) {
            super.onOpen(db);
            new PopulateDbAsync(sInstance).execute();
        }
    };

    private static class PopulateDbAsync extends AsyncTask<Void, Void, Void> {
        private final ConfigurationDao configurationDao;
        private final ConfigurationGroupDao groupDao;

        PopulateDbAsync(AppDatabase db) {
            this.configurationDao = db.configurationDao();
            this.groupDao = db.configurationGroupDao();
        }

        @Override
        protected Void doInBackground(Void... voids) {
            configurationDao.deleteAll();
            groupDao.deleteAll();

            List<ConfigurationGroup> groups = new ArrayList<>();
            groups.add(new ConfigurationGroup(0, "Home", false));
            groups.add(new ConfigurationGroup(0, "Work", false));
            groups.add(new ConfigurationGroup(0, "Lab", false));
            groups.add(new ConfigurationGroup(0, "Classroom", false));
            groups.add(new ConfigurationGroup(0, "Campus", false));

            long[] ids = groupDao.insert(groups);

            int groupId = (int) ids[0];
            List<Configuration> configurations = new ArrayList<>();
            for (int i = 1; i <= 5; i++) {
                String ip = "192.168.0." + (i + 1);
                String name = "Relay " + i;
                configurations.add(new Configuration(0, name, ip, 7000, 8090, i, groupId));
            }

            configurationDao.insert(configurations);

            return null;
        }
    }
}
