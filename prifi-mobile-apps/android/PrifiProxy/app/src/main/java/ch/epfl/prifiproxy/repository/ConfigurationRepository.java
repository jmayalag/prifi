package ch.epfl.prifiproxy.repository;

import android.app.Application;
import android.arch.lifecycle.LiveData;
import android.os.AsyncTask;

import java.util.List;

import ch.epfl.prifiproxy.persistence.AppDatabase;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationDao;
import ch.epfl.prifiproxy.persistence.entity.Configuration;

public class ConfigurationRepository {
    private ConfigurationDao configurationDao;
    private static ConfigurationRepository sInstance;

    public static ConfigurationRepository getInstance(Application application) {
        if (sInstance == null) {
            synchronized (ConfigurationRepository.class) {
                if (sInstance == null) {
                    sInstance = new ConfigurationRepository(application);
                }
            }
        }
        return sInstance;
    }

    ConfigurationRepository(Application application) {
        AppDatabase db = AppDatabase.getDatabase(application);
        configurationDao = db.configurationDao();
    }

    public Configuration getActive() {
        return configurationDao.getActive();
    }

    public LiveData<Configuration> getActiveLive() {
        return configurationDao.getActiveLive();
    }

    public LiveData<Configuration> getConfiguration(int configurationId) {
        return configurationDao.get(configurationId);
    }

    public LiveData<List<Configuration>> getConfigurations(int groupId) {
        return configurationDao.getForGroup(groupId);
    }

    public void insert(Configuration configuration) {
        new InsertAsyncTask(configurationDao).execute(configuration);
    }

    public void update(Configuration configuration) {
        new UpdateAsyncTask(configurationDao).execute(configuration);
    }

    public void update(List<Configuration> configurations) {
        new UpdateAsyncTask(configurationDao)
                .execute(configurations.toArray(new Configuration[configurations.size()]));
    }

    public void delete(Configuration... configuration) {
        new DeleteAsyncTask(configurationDao).execute(configuration);
    }

    private static class InsertAsyncTask extends AsyncTask<Configuration, Void, Void> {
        private final ConfigurationDao dao;

        InsertAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(final Configuration... configurations) {
            if (configurations.length != 1)
                throw new IllegalArgumentException("Must insert one item at a time");

            int count = dao.countConfigurationsForGroups(configurations[0].getGroupId());
            configurations[0].setPriority(count + 1);

            dao.insert(configurations);
            return null;
        }
    }

    private static class UpdateAsyncTask extends AsyncTask<Configuration, Void, Void> {
        private final ConfigurationDao dao;

        UpdateAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(Configuration... configurations) {
            dao.update(configurations);
            return null;
        }
    }

    private static class DeleteAsyncTask extends AsyncTask<Configuration, Void, Void> {
        private final ConfigurationDao dao;

        DeleteAsyncTask(ConfigurationDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(Configuration... configurations) {
            dao.delete(configurations);
            return null;
        }
    }
}