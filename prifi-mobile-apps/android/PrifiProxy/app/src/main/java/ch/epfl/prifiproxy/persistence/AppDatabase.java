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

import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationDao;

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

    private static RoomDatabase.Callback sRoomDatabaseCallback = new RoomDatabase.Callback() {
        @Override
        public void onOpen(@NonNull SupportSQLiteDatabase db) {
            super.onOpen(db);
            new PopulateDbAsync(sInstance).execute();
        }
    };

    private static class PopulateDbAsync extends AsyncTask<Void, Void, Void> {
        private final ConfigurationDao dao;

        PopulateDbAsync(AppDatabase db) {
            this.dao = db.configurationDao();
        }

        @Override
        protected Void doInBackground(Void... voids) {
            dao.deleteAllConfigurations();
            dao.deleteAllConfigurationGroups();

            List<ConfigurationGroup> groups = new ArrayList<>();
            groups.add(new ConfigurationGroup(0, "Home", false));
            groups.add(new ConfigurationGroup(0, "Work", false));
            groups.add(new ConfigurationGroup(0, "Lab", false));
            groups.add(new ConfigurationGroup(0, "Classroom", false));
            groups.add(new ConfigurationGroup(0, "Campus", false));

            dao.insertConfigurationGroups(groups);

            Log.i("APP_DATABASE", "Home id:" + groups.get(0).getId());

            return null;
        }
    }
}
