package ch.epfl.prifiproxy.repository;

import android.app.Application;
import android.arch.lifecycle.LiveData;
import android.os.AsyncTask;
import android.support.annotation.NonNull;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import ch.epfl.prifiproxy.persistence.AppDatabase;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationDao;
import ch.epfl.prifiproxy.persistence.dao.ConfigurationGroupDao;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class ConfigurationGroupRepository {
    private static ConfigurationGroupRepository sInstance;
    private ConfigurationGroupDao groupDao;
    private LiveData<List<ConfigurationGroup>> allGroups;

    public static ConfigurationGroupRepository getInstance(Application application) {
        if (sInstance == null) {
            synchronized (ConfigurationGroupRepository.class) {
                if (sInstance == null) {
                    sInstance = new ConfigurationGroupRepository(application);
                }
            }
        }
        return sInstance;
    }

    private ConfigurationGroupRepository(Application application) {
        AppDatabase db = AppDatabase.getDatabase(application);
        groupDao = db.configurationGroupDao();
        allGroups = groupDao.getAll();
    }

    public void setActiveGroup(ConfigurationGroup group, boolean isActive) {
        group.setActive(isActive);
        new UpdateActiveGroupTask(groupDao, group).execute();
    }

    public LiveData<ConfigurationGroup> getGroup(int groupId) {
        return groupDao.get(groupId);
    }

    public LiveData<List<ConfigurationGroup>> getAllGroups() {
        return allGroups;
    }

    public void insert(ConfigurationGroup group) {
        new InsertAsyncTask(groupDao).execute(group);
    }

    public void update(List<ConfigurationGroup> groups) {
        new UpdateAsyncTask(groupDao)
                .execute(groups.toArray(new ConfigurationGroup[groups.size()]));
    }

    public void delete(List<ConfigurationGroup> groups) {
        new DeleteAsyncTask(groupDao)
                .execute(groups.toArray(new ConfigurationGroup[groups.size()]));
    }

    private static class InsertAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationGroupDao dao;

        InsertAsyncTask(ConfigurationGroupDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(final ConfigurationGroup... configurationGroups) {
            dao.insert(configurationGroups);
            return null;
        }
    }

    private static class UpdateAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationGroupDao dao;

        UpdateAsyncTask(ConfigurationGroupDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(ConfigurationGroup... groups) {
            dao.update(groups);
            return null;
        }
    }

    private static class DeleteAsyncTask extends AsyncTask<ConfigurationGroup, Void, Void> {
        private final ConfigurationGroupDao dao;

        DeleteAsyncTask(ConfigurationGroupDao dao) {
            this.dao = dao;
        }

        @Override
        protected Void doInBackground(ConfigurationGroup... groups) {
            dao.delete(groups);
            return null;
        }
    }

    private static class UpdateActiveGroupTask extends AsyncTask<Void, Void, Void> {
        private final ConfigurationGroupDao dao;
        private final ConfigurationGroup changed;

        public UpdateActiveGroupTask(ConfigurationGroupDao dao, @NonNull ConfigurationGroup changed) {
            this.dao = dao;
            this.changed = changed;
        }

        @Override
        protected Void doInBackground(Void... voids) {
            ConfigurationGroup currentActive = dao.getActive();
            List<ConfigurationGroup> updates = new ArrayList<>();
            updates.add(changed);

            // Deactivate current active group
            if (changed.isActive() && currentActive != null) {
                currentActive.setActive(false);
                updates.add(currentActive);
            }

            dao.update(updates);
            return null;
        }
    }
}
